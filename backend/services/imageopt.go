package services

import (
	"fmt"
	"image"
	_ "image/gif" // decode support
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// Image optimization for uploads. Photos uploaded from phones are routinely
// 2–4MB; the storefront was serving them verbatim, which made every page with
// imagery painfully slow. All admin/storefront upload paths now pass through
// SaveOptimizedImage, and cmd/imgopt shrinks the files that already exist.

const (
	// JPEGQuality balances size vs. artifacts for product photography.
	JPEGQuality = 82
	// MaxDimProduct bounds product / sale-page / slip images (long side, px).
	MaxDimProduct = 1600
	// MaxDimSite bounds full-bleed site images (hero, lookbook) — larger
	// because they render viewport-wide.
	MaxDimSite = 2000
)

type opaquer interface{ Opaque() bool }

// isOpaque reports whether an image can safely be flattened to JPEG. Photos
// exported from design tools often carry an alpha channel with only a handful
// of translucent edge pixels — treating those as "transparent" would leave
// multi-MB PNGs, so a negligible alpha fraction (<0.5% of sampled pixels)
// still counts as opaque; such images are composited onto white before the
// JPEG encode (see flattenOnWhite).
func isOpaque(img image.Image) bool {
	if o, ok := img.(opaquer); ok && o.Opaque() {
		return true
	}
	// Sample the alpha channel on a coarse grid.
	b := img.Bounds()
	var sampled, translucent int
	for y := b.Min.Y; y < b.Max.Y; y += 8 {
		for x := b.Min.X; x < b.Max.X; x += 8 {
			sampled++
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xf000 {
				translucent++
			}
		}
	}
	if sampled == 0 {
		return true
	}
	return float64(translucent)/float64(sampled) < 0.005
}

// flattenOnWhite composites img onto a white background so a stray alpha
// channel doesn't turn into black fringes in the JPEG encode.
func flattenOnWhite(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	xdraw.Draw(dst, dst.Bounds(), image.White, image.Point{}, xdraw.Src)
	xdraw.Draw(dst, dst.Bounds(), img, b.Min, xdraw.Over)
	return dst
}

// resizeMax scales img down so its long side is at most maxDim (never up).
func resizeMax(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if (w <= maxDim && h <= maxDim) || w == 0 || h == 0 {
		return img
	}
	scale := float64(maxDim) / float64(w)
	if h > w {
		scale = float64(maxDim) / float64(h)
	}
	nw := int(float64(w)*scale + 0.5)
	nh := int(float64(h)*scale + 0.5)
	dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst
}

func copyMultipart(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// SaveOptimizedImage stores an uploaded image under dir as base+ext, resized
// to fit maxDim and re-encoded (JPEG for opaque images, PNG when transparency
// must survive). Formats we cannot decode — and GIFs, whose animation a
// re-encode would destroy — are stored as received with their original
// extension. Returns the final filename (base + chosen extension).
func SaveOptimizedImage(fh *multipart.FileHeader, dir, base string, maxDim int) (string, error) {
	origExt := strings.ToLower(filepath.Ext(fh.Filename))
	if origExt == ".gif" {
		name := base + origExt
		return name, copyMultipart(fh, filepath.Join(dir, name))
	}

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	img, _, decErr := image.Decode(src)
	src.Close()
	if decErr != nil {
		// Not decodable (e.g. webp/heic) — store as received.
		name := base + origExt
		return name, copyMultipart(fh, filepath.Join(dir, name))
	}

	img = resizeMax(img, maxDim)
	ext := ".jpg"
	if !isOpaque(img) {
		ext = ".png"
	}
	name := base + ext
	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	if ext == ".jpg" {
		err = jpeg.Encode(out, flattenOnWhite(img), &jpeg.Options{Quality: JPEGQuality})
	} else {
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(out, img)
	}
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return name, nil
}

// OptimizeFileInPlace shrinks an existing image file, keeping its filename and
// format (DB rows reference the name). The file is only replaced when the
// optimized version is actually smaller. Returns old and new sizes in bytes.
func OptimizeFileInPlace(path string, maxDim int) (int64, int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	oldSize := fi.Size()

	f, err := os.Open(path)
	if err != nil {
		return oldSize, oldSize, err
	}
	img, format, err := image.Decode(f)
	f.Close()
	if err != nil {
		return oldSize, oldSize, fmt.Errorf("decode: %w", err)
	}
	if format != "jpeg" && format != "png" {
		return oldSize, oldSize, fmt.Errorf("unsupported format %q", format)
	}

	img = resizeMax(img, maxDim)
	tmp := path + ".opt"
	out, err := os.Create(tmp)
	if err != nil {
		return oldSize, oldSize, err
	}
	if format == "jpeg" {
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: JPEGQuality})
	} else {
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(out, img)
	}
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return oldSize, oldSize, err
	}

	ni, err := os.Stat(tmp)
	if err != nil {
		os.Remove(tmp)
		return oldSize, oldSize, err
	}
	if ni.Size() >= oldSize {
		os.Remove(tmp)
		return oldSize, oldSize, nil
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return oldSize, oldSize, err
	}
	return oldSize, ni.Size(), nil
}

// ConvertPNGToJPEG re-encodes an opaque PNG photo as a JPEG sibling
// (same basename, .jpg extension), resized to fit maxDim. Returns the new
// path and true when a JPEG was written; PNGs with real transparency are
// left alone. The caller is responsible for updating any DB references.
func ConvertPNGToJPEG(path string, maxDim int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	img, format, err := image.Decode(f)
	f.Close()
	if err != nil {
		return "", false, err
	}
	if format != "png" || !isOpaque(img) {
		return "", false, nil
	}
	img = resizeMax(img, maxDim)
	newPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
	out, err := os.Create(newPath)
	if err != nil {
		return "", false, err
	}
	err = jpeg.Encode(out, flattenOnWhite(img), &jpeg.Options{Quality: JPEGQuality})
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(newPath)
		return "", false, err
	}
	return newPath, true, nil
}
