package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const vendingMediaMaxSize int64 = 100 << 20

type vendingMediaImportReq struct {
	URL  string `json:"url" binding:"required"`
	Type string `json:"type"`
}

func vendingMediaDir() string {
	return filepath.Join(config.C.UploadDir, "vending")
}

func vendingMediaRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func mediaFilename(prefix, ext string) string {
	return fmt.Sprintf("%s%d_%s%s", prefix, time.Now().UnixNano(), vendingMediaRandomHex(4), ext)
}

func vendingExtensionFromContentType(ct string) string {
	switch strings.ToLower(strings.Split(ct, ";")[0]) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	case "video/x-matroska":
		return ".mkv"
	case "application/vnd.apple.mpegurl", "application/x-mpegurl":
		return ".m3u8"
	}
	return ""
}

func vendingMediaType(ct, name string) string {
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	if ct == "image/svg+xml" {
		return ""
	}
	switch {
	case strings.HasPrefix(ct, "image/"):
		return "image"
	case strings.HasPrefix(ct, "video/"):
		return "video"
	case strings.Contains(ct, "mpegurl"):
		return "video"
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		return "image"
	case ".mp4", ".webm", ".mov", ".m4v", ".avi", ".mkv", ".m3u8":
		return "video"
	}
	return ""
}

func vendingMediaExt(ct, name string) string {
	if ext := vendingExtensionFromContentType(ct); ext != "" {
		return ext
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".mp4", ".webm", ".mov", ".m4v", ".avi", ".mkv", ".m3u8":
		return ext
	}
	return ""
}

func vendingIsDownloaderHost(host string) bool {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	if host == "youtube.com" || host == "youtu.be" || host == "m.youtube.com" || strings.HasSuffix(host, ".youtube.com") {
		return true
	}
	for _, h := range []string{
		"instagram.com", "instagr.am", "tiktok.com", "vm.tiktok.com",
		"facebook.com", "fb.watch", "twitter.com", "x.com",
		"vimeo.com", "dailymotion.com", "dai.ly", "twitch.tv",
	} {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
}

func vendingYtDlpPath() string {
	if p := strings.TrimSpace(os.Getenv("YT_DLP_PATH")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p
	}
	for _, p := range []string{
		"/home/andtho89/.local/bin/yt-dlp",
		"/usr/local/bin/yt-dlp",
		filepath.Join("bin", "yt-dlp"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func vendingFfmpegPath() string {
	if p := strings.TrimSpace(os.Getenv("FFMPEG_PATH")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	for _, p := range []string{
		"/home/andtho89/.local/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		filepath.Join("bin", "ffmpeg"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func vendingNodePath() string {
	if p, err := exec.LookPath("node"); err == nil {
		return p
	}
	matches, _ := filepath.Glob("/home/andtho89/.nvm/versions/node/*/bin/node")
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return matches[len(matches)-1]
}

func vendingDownloadWithYtDlp(rawURL string) (typ, publicURL, checksum string, err error) {
	return downloadWithYtDlpTo(vendingMediaDir(), "/uploads/vending", "vm_", rawURL)
}

func downloadWithYtDlpTo(destDir, publicBase, prefix, rawURL string) (typ, publicURL, checksum string, err error) {
	bin := vendingYtDlpPath()
	if bin == "" {
		return "", "", "", fmt.Errorf("yt-dlp belum tersedia di server")
	}
	tmpDir, err := os.MkdirTemp("", "ttpg-ytdlp-*")
	if err != nil {
		return "", "", "", err
	}
	defer os.RemoveAll(tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	args := []string{
		"--no-playlist", "--no-progress", "--max-filesize", "100M",
		"-o", filepath.Join(tmpDir, "media.%(ext)s"),
	}
	if ff := vendingFfmpegPath(); ff != "" {
		args = append(args, "--ffmpeg-location", ff,
			"-f", "bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b",
			"--merge-output-format", "mp4")
	} else {
		args = append(args, "-f", "b[ext=mp4]/b")
	}
	if node := vendingNodePath(); node != "" {
		args = append(args, "--js-runtimes", "node:"+node)
	}
	args = append(args, rawURL)
	cmd := exec.CommandContext(ctx, bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		msg = strings.ReplaceAll(msg, "\n", " ")
		if len(msg) > 280 {
			msg = msg[:280]
		}
		return "", "", "", fmt.Errorf("gagal mengunduh media: %s", msg)
	}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", "", "", err
	}
	var chosen string
	var chosenSize int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() > chosenSize {
			chosenSize = info.Size()
			chosen = filepath.Join(tmpDir, e.Name())
		}
	}
	if chosen == "" {
		return "", "", "", fmt.Errorf("file hasil unduhan tidak ditemukan")
	}
	if chosenSize > vendingMediaMaxSize {
		return "", "", "", fmt.Errorf("ukuran media melebihi 100MB")
	}
	f, err := os.Open(chosen)
	if err != nil {
		return "", "", "", err
	}
	defer f.Close()
	ext := strings.ToLower(filepath.Ext(chosen))
	ct := "video/mp4"
	if ext == ".webm" {
		ct = "video/webm"
	} else if ext == ".mkv" {
		ct = "video/x-matroska"
	} else if ext == ".mov" {
		ct = "video/quicktime"
	}
	return saveMediaFileTo(destDir, publicBase, prefix, f, filepath.Base(chosen), ct)
}

func vendingSaveMedia(r io.Reader, name, ct string) (typ, publicURL, checksum string, err error) {
	return saveMediaFileTo(vendingMediaDir(), "/uploads/vending", "vm_", r, name, ct)
}

func saveMediaFileTo(destDir, publicBase, prefix string, r io.Reader, name, ct string) (typ, publicURL, checksum string, err error) {
	typ = vendingMediaType(ct, name)
	if typ == "" {
		return "", "", "", fmt.Errorf("format media tidak didukung (hanya gambar/video)")
	}
	ext := vendingMediaExt(ct, name)
	if ext == "" {
		ext = ".bin"
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", "", err
	}
	filename := mediaFilename(prefix, ext)
	dest := filepath.Join(destDir, filename)
	f, err := os.Create(dest)
	if err != nil {
		return "", "", "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, vendingMediaMaxSize+1))
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(dest)
		return "", "", "", copyErr
	}
	if closeErr != nil {
		os.Remove(dest)
		return "", "", "", closeErr
	}
	if n > vendingMediaMaxSize {
		os.Remove(dest)
		return "", "", "", fmt.Errorf("ukuran media melebihi 100MB")
	}
	if n == 0 {
		os.Remove(dest)
		return "", "", "", fmt.Errorf("file media kosong")
	}
	return typ, strings.TrimRight(publicBase, "/") + "/" + url.PathEscape(filename), hex.EncodeToString(h.Sum(nil)), nil
}

func vendingInsertMedia(typ, mediaURL, checksum string) (int64, error) {
	res, err := db.DB.Exec("INSERT INTO vending_media (type, url, `order`, checksum, enabled) VALUES (?, ?, 0, ?, 1)", typ, mediaURL, checksum)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func VendingAdminMediaList(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	rows, err := db.DB.Query("SELECT id, type, url, `order`, COALESCE(checksum, ''), enabled, created_at FROM vending_media ORDER BY id DESC")
	if err != nil {
		response.Error(c, 500, "gagal membaca media: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var (
			id, order, enabled      int
			typ, mediaURL, checksum string
			createdAt               time.Time
		)
		if err := rows.Scan(&id, &typ, &mediaURL, &order, &checksum, &enabled, &createdAt); err != nil {
			continue
		}
		list = append(list, map[string]any{
			"id":         id,
			"type":       typ,
			"url":        mediaURL,
			"order":      order,
			"checksum":   checksum,
			"enabled":    enabled == 1,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}
	response.Success(c, 200, "Daftar media vending", list, nil)
}

func VendingAdminMediaUpload(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, vendingMediaMaxSize+(1<<20))
	file, hdr, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, 400, "file wajib diunggah", nil)
		return
	}
	defer file.Close()
	typ, mediaURL, checksum, err := vendingSaveMedia(file, hdr.Filename, hdr.Header.Get("Content-Type"))
	if err != nil {
		response.Error(c, 400, err.Error(), nil)
		return
	}
	id, err := vendingInsertMedia(typ, mediaURL, checksum)
	if err != nil {
		response.Error(c, 500, "gagal menyimpan media: "+err.Error(), nil)
		return
	}
	response.Success(c, 201, "Media berhasil diunggah", gin.H{
		"id": id, "type": typ, "url": mediaURL, "checksum": checksum,
	}, nil)
}

func VendingAdminMediaImport(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	var req vendingMediaImportReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		response.Error(c, 400, "url wajib diisi", nil)
		return
	}
	rawURL := strings.TrimSpace(req.URL)
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		response.Error(c, 400, "url tidak valid", nil)
		return
	}
	if vendingIsDownloaderHost(u.Host) {
		typ, mediaURL, checksum, err := vendingDownloadWithYtDlp(rawURL)
		if err != nil {
			response.Error(c, 400, err.Error(), nil)
			return
		}
		id, err := vendingInsertMedia(typ, mediaURL, checksum)
		if err != nil {
			response.Error(c, 500, "gagal menyimpan media: "+err.Error(), nil)
			return
		}
		response.Success(c, 201, "Media berhasil diunduh", gin.H{
			"id": id, "type": typ, "url": mediaURL, "checksum": checksum,
		}, nil)
		return
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		response.Error(c, 400, "gagal mengunduh url: "+err.Error(), nil)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Error(c, 400, fmt.Sprintf("url mengembalikan status %d", resp.StatusCode), nil)
		return
	}
	if resp.ContentLength > vendingMediaMaxSize {
		response.Error(c, 400, "ukuran media melebihi 100MB", nil)
		return
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(ct), "text/html") || strings.Contains(strings.ToLower(ct), "application/xhtml") {
		id, err := vendingInsertMedia("website", rawURL, "")
		if err != nil {
			response.Error(c, 500, "gagal menyimpan media: "+err.Error(), nil)
			return
		}
		response.Success(c, 201, "URL website disimpan", gin.H{"id": id, "type": "website", "url": rawURL}, nil)
		return
	}
	typ, mediaURL, checksum, err := vendingSaveMedia(resp.Body, filepath.Base(u.Path), ct)
	if err != nil {
		response.Error(c, 400, err.Error(), nil)
		return
	}
	id, err := vendingInsertMedia(typ, mediaURL, checksum)
	if err != nil {
		response.Error(c, 500, "gagal menyimpan media: "+err.Error(), nil)
		return
	}
	response.Success(c, 201, "Media berhasil diunduh", gin.H{
		"id": id, "type": typ, "url": mediaURL, "checksum": checksum,
	}, nil)
}

func VendingAdminMediaDelete(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	var mediaURL string
	if err := db.DB.QueryRow("SELECT url FROM vending_media WHERE id = ?", id).Scan(&mediaURL); err != nil {
		response.Error(c, 404, "media tidak ditemukan", nil)
		return
	}
	var used int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM vending_playlist_items WHERE media_id = ?", id).Scan(&used); err != nil {
		response.Error(c, 500, "gagal memeriksa pemakaian media: "+err.Error(), nil)
		return
	}
	if used > 0 {
		response.Error(c, 409, "media masih dipakai item playlist", nil)
		return
	}
	if _, err := db.DB.Exec("DELETE FROM vending_media WHERE id = ?", id); err != nil {
		response.Error(c, 500, "gagal hapus media: "+err.Error(), nil)
		return
	}
	if strings.HasPrefix(mediaURL, "/uploads/vending/") {
		os.Remove(filepath.Join(vendingMediaDir(), filepath.Base(mediaURL)))
	}
	response.Success(c, 200, "Media berhasil dihapus", gin.H{"id": id}, nil)
}
