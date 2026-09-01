// imgopt shrinks existing storefront-facing upload images in place.
//
//	go run ./cmd/imgopt -dir ./uploads [-min 250] [-dry]
//
// Only product_*, site_* and salepage_* files are touched (slips and chat
// media are left alone). Filenames and formats are preserved — DB rows
// reference them — and each original is copied to <dir>_originals/ before
// being replaced, so the run is reversible.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"brunocollective_inventory/services"
)

func backup(src, backupDir, name string) error {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(backupDir, name)
	if _, err := os.Stat(dst); err == nil {
		return nil // already backed up on a previous run
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func main() {
	dir := flag.String("dir", "./uploads", "uploads directory")
	minKB := flag.Int64("min", 250, "only touch files larger than this (KB)")
	dry := flag.Bool("dry", false, "list what would change without writing")
	flag.Parse()

	backupDir := strings.TrimRight(*dir, "/") + "_originals"
	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read dir:", err)
		os.Exit(1)
	}

	var files, changed int
	var savedBytes int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if !(strings.HasPrefix(lower, "product_") ||
			strings.HasPrefix(lower, "site_") ||
			strings.HasPrefix(lower, "salepage_")) {
			continue
		}
		if !(strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") ||
			strings.HasSuffix(lower, ".png")) {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() < *minKB*1024 {
			continue
		}
		files++
		path := filepath.Join(*dir, name)
		maxDim := services.MaxDimProduct
		if strings.HasPrefix(lower, "site_") {
			maxDim = services.MaxDimSite
		}
		if *dry {
			fmt.Printf("would optimize %-60s %6.0f KB\n", name, float64(info.Size())/1024)
			continue
		}
		if err := backup(path, backupDir, name); err != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s: backup failed: %v\n", name, err)
			continue
		}
		oldSize, newSize, err := services.OptimizeFileInPlace(path, maxDim)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s: %v\n", name, err)
			continue
		}
		if newSize < oldSize {
			changed++
			savedBytes += oldSize - newSize
			fmt.Printf("%-60s %6.0f KB -> %5.0f KB\n", name, float64(oldSize)/1024, float64(newSize)/1024)
		}
	}
	fmt.Printf("\n%d candidate files, %d optimized, %.1f MB saved (originals in %s)\n",
		files, changed, float64(savedBytes)/1024/1024, backupDir)
}
