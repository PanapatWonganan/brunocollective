package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/middleware"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
)

// TryOnHandler serves the storefront "ลองใส่ดู" (virtual try-on) feature:
// a shopper uploads a photo of themselves and gets back the same photo with
// the product worn. Public (no login), throttled per day — per IP for guests
// and per customer for members, since every generation costs money.
//
// The customer photo is never persisted: it is normalised in memory, sent to
// the model, and dropped. Only the product image is read from disk.
type TryOnHandler struct {
	Config *config.Config
	Client *services.TryOnClient

	mu     sync.Mutex
	counts map[string]tryOnCount // key = "ip:1.2.3.4" | "member:42"

	jobsMu sync.Mutex
	jobs   map[string]*tryOnJob

	cutoutMu sync.Mutex // serialises garment cutout generation
}

// tryOnJob is one in-flight or finished generation. Generation takes
// 60–90s, longer than nginx/Cloudflare keep a request open, so POST returns
// a job id immediately and the storefront polls GET /api/shop/try-on/jobs/:id.
// Finished jobs are held in memory for tryOnJobTTL and then dropped — the
// result image is never written to disk.
type tryOnJob struct {
	ID        string
	Status    string // pending | done | error
	Image     string // data: URL when done
	Error     string // Thai message when error
	HTTPCode  int    // status code to report with Error
	Remaining int
	Created   time.Time
	Done      time.Time
}

const tryOnJobTTL = 10 * time.Minute

type tryOnCount struct {
	day string // YYYY-MM-DD (Asia/Bangkok) the count belongs to
	n   int
}

const (
	tryOnMaxUpload   = 8 << 20 // raw upload cap (before normalisation)
	tryOnPhotoMaxDim = 1280    // long side sent to the model
)

func NewTryOnHandler(cfg *config.Config, client *services.TryOnClient) *TryOnHandler {
	return &TryOnHandler{Config: cfg, Client: client, counts: map[string]tryOnCount{}, jobs: map[string]*tryOnJob{}}
}

func (h *TryOnHandler) enabled() bool { return h.Client != nil && h.Client.Enabled() }

// quotaKey identifies the caller for the daily limit and returns their limit.
func (h *TryOnHandler) quotaKey(c *fiber.Ctx) (string, int) {
	if id := middleware.OptionalMemberID(c, h.Config); id > 0 {
		return fmt.Sprintf("member:%d", id), h.Config.TryOnMemberDailyLimit
	}
	ip := strings.TrimSpace(strings.Split(c.Get("X-Forwarded-For"), ",")[0])
	if ip == "" {
		ip = c.IP()
	}
	return "ip:" + ip, h.Config.TryOnDailyLimit
}

func bangkokDay() string {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// remaining reports how many generations the caller has left today.
func (h *TryOnHandler) remaining(key string, limit int) int {
	if limit <= 0 {
		return 1 << 30 // unlimited
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	cnt, ok := h.counts[key]
	if !ok || cnt.day != bangkokDay() {
		return limit
	}
	if left := limit - cnt.n; left > 0 {
		return left
	}
	return 0
}

// consume takes one generation from the caller's daily quota; false = exhausted.
func (h *TryOnHandler) consume(key string, limit int) bool {
	if limit <= 0 {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	day := bangkokDay()
	cnt := h.counts[key]
	if cnt.day != day {
		cnt = tryOnCount{day: day}
	}
	if cnt.n >= limit {
		return false
	}
	cnt.n++
	h.counts[key] = cnt
	// Opportunistic sweep so the map doesn't grow forever.
	if len(h.counts) > 5000 {
		for k, v := range h.counts {
			if v.day != day {
				delete(h.counts, k)
			}
		}
	}
	return true
}

func (h *TryOnHandler) refund(key string, limit int) {
	if limit <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if cnt, ok := h.counts[key]; ok && cnt.n > 0 && cnt.day == bangkokDay() {
		cnt.n--
		h.counts[key] = cnt
	}
}

// Status — GET /api/shop/try-on: is the feature on, and how many tries the
// caller has left today. The storefront hides the button when disabled.
func (h *TryOnHandler) Status(c *fiber.Ctx) error {
	if !h.enabled() {
		return c.JSON(fiber.Map{"enabled": false})
	}
	key, limit := h.quotaKey(c)
	return c.JSON(fiber.Map{
		"enabled":   true,
		"remaining": h.remaining(key, limit),
		"limit":     limit,
		"member":    strings.HasPrefix(key, "member:"),
	})
}

// Generate — POST /api/shop/try-on (multipart): photo (image file),
// product_id, optional image (one of the product's image URLs — lets the
// shopper try a specific colourway). Returns {image: "data:...", remaining}.
func (h *TryOnHandler) Generate(c *fiber.Ctx) error {
	if !h.enabled() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "ฟีเจอร์ลองใส่ยังไม่เปิดใช้งาน"})
	}

	productID, _ := strconv.Atoi(c.FormValue("product_id"))
	if productID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "product_id is required"})
	}
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบสินค้า"})
	}

	fh, err := c.FormFile("photo")
	if err != nil || fh == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณาเลือกรูปของคุณก่อน"})
	}
	if fh.Size > tryOnMaxUpload {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รูปใหญ่เกินไป (สูงสุด 8MB)"})
	}
	if ct := fh.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รองรับเฉพาะไฟล์รูปภาพ"})
	}
	src, err := fh.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "อ่านรูปไม่สำเร็จ"})
	}
	raw, err := io.ReadAll(io.LimitReader(src, tryOnMaxUpload+1))
	src.Close()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "อ่านรูปไม่สำเร็จ"})
	}
	photo, err := services.NormalizeJPEG(raw, tryOnPhotoMaxDim)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "อ่านรูปไม่สำเร็จ — ลองใช้ไฟล์ JPG หรือ PNG"})
	}

	// Garment image: the requested gallery image when it belongs to this
	// product, else the primary image.
	garmentURL := pickGarmentURL(&product, c.FormValue("image"))
	garment, garmentMime, err := h.garmentImage(context.Background(), &product, garmentURL)
	if err != nil {
		log.Printf("try-on: product %d image %q: %v", product.ID, garmentURL, err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "สินค้านี้ยังไม่มีรูปสำหรับลองใส่"})
	}

	key, limit := h.quotaKey(c)
	if !h.consume(key, limit) {
		msg := "วันนี้ลองใส่ครบจำนวนแล้ว พรุ่งนี้ลองใหม่ได้อีกครั้ง"
		if strings.HasPrefix(key, "ip:") {
			msg = "วันนี้ลองใส่ครบจำนวนแล้ว — สมัครสมาชิกเพื่อลองได้มากขึ้น หรือกลับมาใหม่พรุ่งนี้"
		}
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": msg, "remaining": 0})
	}

	job := &tryOnJob{ID: newTryOnJobID(), Status: "pending", Created: time.Now(), Remaining: h.remaining(key, limit)}
	h.jobsMu.Lock()
	h.jobs[job.ID] = job
	h.sweepJobsLocked()
	h.jobsMu.Unlock()

	productID64, productName, productCategory := product.ID, product.Name, product.Category
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 175*time.Second)
		defer cancel()
		started := time.Now()
		res, err := h.Client.Generate(ctx, photo, "image/jpeg", garment, garmentMime, productName, productCategory)
		h.jobsMu.Lock()
		defer h.jobsMu.Unlock()
		job.Done = time.Now()
		if err != nil {
			h.refund(key, limit)
			job.Remaining = h.remaining(key, limit)
			job.Status = "error"
			log.Printf("try-on: product %d failed after %s: %v", productID64, time.Since(started).Round(time.Millisecond), err)
			switch {
			case errors.Is(err, services.ErrTryOnBusy):
				job.HTTPCode, job.Error = fiber.StatusTooManyRequests, "ตอนนี้มีคนลองใส่พร้อมกันเยอะ กรุณาลองใหม่ในอีกสักครู่"
			case errors.Is(err, services.ErrTryOnNoImage):
				job.HTTPCode, job.Error = fiber.StatusUnprocessableEntity, "สร้างภาพไม่สำเร็จ ลองใช้รูปที่เห็นตัวชัด ๆ แสงสว่าง และไม่มีคนอื่นในรูป"
			default:
				job.HTTPCode, job.Error = fiber.StatusBadGateway, "ระบบลองใส่ขัดข้องชั่วคราว กรุณาลองใหม่อีกครั้ง"
			}
			return
		}
		log.Printf("try-on: product %d ok in %s (%s, %d bytes, %s)", productID64, time.Since(started).Round(time.Millisecond), key, len(res.Data), res.Mime)
		job.Status = "done"
		job.Image = "data:" + res.Mime + ";base64," + base64.StdEncoding.EncodeToString(res.Data)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"job_id": job.ID, "remaining": job.Remaining})
}

// Job — GET /api/shop/try-on/jobs/:id: poll a generation. 202 while pending,
// 200 with the image when done, the mapped error status when it failed,
// 404 when unknown/expired.
func (h *TryOnHandler) Job(c *fiber.Ctx) error {
	h.jobsMu.Lock()
	job, ok := h.jobs[c.Params("id")]
	var snap tryOnJob
	if ok {
		snap = *job
	}
	h.jobsMu.Unlock()
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบงานนี้ กรุณาลองใหม่"})
	}
	switch snap.Status {
	case "done":
		return c.JSON(fiber.Map{"status": "done", "image": snap.Image, "remaining": snap.Remaining})
	case "error":
		return c.Status(snap.HTTPCode).JSON(fiber.Map{"status": "error", "error": snap.Error, "remaining": snap.Remaining})
	default:
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "pending", "elapsed_ms": time.Since(snap.Created).Milliseconds()})
	}
}

func newTryOnJobID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}

// sweepJobsLocked drops finished jobs older than the TTL and pending ones
// that are absurdly old (a goroutine that never returned). Caller holds jobsMu.
func (h *TryOnHandler) sweepJobsLocked() {
	now := time.Now()
	for id, j := range h.jobs {
		if (j.Status != "pending" && now.Sub(j.Done) > tryOnJobTTL) || now.Sub(j.Created) > 3*tryOnJobTTL {
			delete(h.jobs, id)
		}
	}
}

// pickGarmentURL returns the requested gallery image when it belongs to the
// product, else the primary image.
func pickGarmentURL(product *models.Product, want string) string {
	want = strings.TrimSpace(want)
	if want != "" {
		if want == product.ImageURL {
			return want
		}
		for _, u := range product.Images {
			if u == want {
				return want
			}
		}
	}
	if product.ImageURL != "" {
		return product.ImageURL
	}
	if len(product.Images) > 0 {
		return product.Images[0]
	}
	return ""
}

// garmentCachePath is where the clean cutout of a product image lives.
func (h *TryOnHandler) garmentCachePath(url string) string {
	sum := sha1.Sum([]byte(url))
	return filepath.Join(h.Config.UploadDir, "tryon_garment_"+hex.EncodeToString(sum[:8])+".jpg")
}

// garmentImage returns the clean "item only" reference for a product image:
// the cached cutout when present, else it generates one (once per image),
// falling back to the raw product photo if generation fails.
func (h *TryOnHandler) garmentImage(ctx context.Context, product *models.Product, url string) ([]byte, string, error) {
	if url == "" {
		return nil, "", errors.New("no image")
	}
	cache := h.garmentCachePath(url)
	if data, err := os.ReadFile(cache); err == nil && len(data) > 0 {
		return data, "image/jpeg", nil
	}
	raw, mime, err := h.loadProductImage(url)
	if err != nil {
		return nil, "", err
	}
	h.cutoutMu.Lock() // one cutout at a time (cost + rate limits)
	defer h.cutoutMu.Unlock()
	if data, err := os.ReadFile(cache); err == nil && len(data) > 0 {
		return data, "image/jpeg", nil // generated while we waited
	}
	cctx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	started := time.Now()
	res, err := h.Client.Cutout(cctx, raw, mime, product.Name, product.Category)
	if err != nil {
		log.Printf("try-on: cutout for %q failed after %s (using raw photo): %v", url, time.Since(started).Round(time.Millisecond), err)
		return raw, mime, nil
	}
	jpg, err := services.NormalizeJPEG(res.Data, tryOnPhotoMaxDim)
	if err != nil {
		return raw, mime, nil
	}
	if err := os.WriteFile(cache, jpg, 0o644); err != nil {
		log.Printf("try-on: cutout cache write: %v", err)
	}
	log.Printf("try-on: cutout for product %d %q in %s", product.ID, url, time.Since(started).Round(time.Millisecond))
	return jpg, "image/jpeg", nil
}

// Garment — GET /api/shop/products/:id/garment?image=: the clean cutout of a
// product image (generated on first request). Used by the live try-on as
// the reference image Decart sees.
func (h *TryOnHandler) Garment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบสินค้า"})
	}
	url := pickGarmentURL(&product, c.Query("image"))
	var data []byte
	var mime string
	var err error
	if h.enabled() {
		data, mime, err = h.garmentImage(context.Background(), &product, url)
	} else {
		data, mime, err = h.loadProductImage(url)
	}
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "สินค้านี้ยังไม่มีรูป"})
	}
	c.Set("Content-Type", mime)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(data)
}

// WarmGarments pre-generates cutouts for every product image in the
// background (one at a time, cached on disk, so it only costs on first run
// per image) so shoppers never wait for the extra step.
func (h *TryOnHandler) WarmGarments() {
	if !h.enabled() {
		return
	}
	var products []models.Product
	database.DB.Order("display_order ASC, id ASC").Find(&products)
	n := 0
	for i := range products {
		p := &products[i]
		urls := append([]string{}, p.ImageURL)
		urls = append(urls, p.Images...)
		for _, u := range urls {
			if u == "" {
				continue
			}
			if _, err := os.Stat(h.garmentCachePath(u)); err == nil {
				continue
			}
			if _, _, err := h.garmentImage(context.Background(), p, u); err == nil {
				n++
			}
			time.Sleep(2 * time.Second)
		}
	}
	if n > 0 {
		log.Printf("try-on: warmed %d garment cutouts", n)
	}
}

// loadProductImage reads a product image (an /uploads/… path from the upload
// dir, or an absolute URL fetched over HTTP) and returns bytes + mime.
func (h *TryOnHandler) loadProductImage(url string) ([]byte, string, error) {
	if url == "" {
		return nil, "", errors.New("no image")
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		resp, err := (&http.Client{Timeout: 20 * time.Second}).Get(url)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("http %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		if err != nil {
			return nil, "", err
		}
		return normalizeGarment(data)
	}
	// "/uploads/x.jpg" → <UploadDir>/x.jpg; base name only, no traversal.
	name := filepath.Base(url)
	data, err := os.ReadFile(filepath.Join(h.Config.UploadDir, name))
	if err != nil {
		return nil, "", err
	}
	return normalizeGarment(data)
}

// normalizeGarment re-encodes the product image as a bounded JPEG so the
// request stays small; undecodable formats (webp) are passed through as-is.
func normalizeGarment(data []byte) ([]byte, string, error) {
	if jpg, err := services.NormalizeJPEG(data, tryOnPhotoMaxDim); err == nil {
		return jpg, "image/jpeg", nil
	}
	return data, http.DetectContentType(data), nil
}
