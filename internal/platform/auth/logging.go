package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
)

const maxLogBody = 16 << 10
const maxLogParams = 4000

var sensitiveJSONKeys = map[string]bool{
	"password": true, "passwd": true, "pass": true, "secret": true,
	"secret_key": true, "client_secret": true, "api_key": true, "apikey": true,
	"token": true, "access_token": true, "refresh_token": true, "authorization": true,
	"x-ttpg-key": true, "x-api-key": true, "private_key": true,
}

var sensitiveKVRe = regexp.MustCompile(`(?i)("?\b(?:password|passwd|secret|secret_key|api_key|apikey|token|access_token|refresh_token|authorization|private_key)"?\s*[:=]\s*)"?[^"&,\s}]+`)

func redactString(s string) string {
	return sensitiveKVRe.ReplaceAllString(s, "${1}[REDACTED]")
}

func redactValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if sensitiveJSONKeys[strings.ToLower(k)] {
				t[k] = "[REDACTED]"
				continue
			}
			redactValue(val)
		}
	case []any:
		for _, item := range t {
			redactValue(item)
		}
	}
}

// redactBody mengembalikan salinan body yang sudah aman disimpan ke log (rahasia disembunyikan).
func redactBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	if s[0] == '{' || s[0] == '[' {
		var v any
		if err := json.Unmarshal(b, &v); err == nil {
			redactValue(v)
			if out, err := json.Marshal(v); err == nil {
				return string(out)
			}
		}
	}
	return redactString(s)
}

// captureBody menyimpan body (sudah di-redact) untuk log dan mengembalikan body asli ke handler.
func captureBody(c *gin.Context) {
	if c.Request.Body == nil || c.Request.Method == "GET" || c.Request.Method == "HEAD" {
		return
	}
	b, _ := io.ReadAll(io.LimitReader(c.Request.Body, maxLogBody))
	c.Request.Body = io.NopCloser(bytes.NewReader(b))
	ct := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(ct, "multipart/form-data") || strings.Contains(ct, "application/octet-stream") {
		c.Set("req_body", "")
		return
	}
	c.Set("req_body", redactBody(b))
}

func logParams(c *gin.Context) string {
	query := redactString(c.Request.URL.RawQuery)
	body, _ := c.Get("req_body")
	bodyStr, _ := body.(string)
	out, err := json.Marshal(map[string]string{"query": query, "body": bodyStr})
	if err != nil {
		return query
	}
	s := string(out)
	if len(s) > maxLogParams {
		s = s[:maxLogParams]
	}
	return s
}

var logRetentionDays atomic.Int64

func SetLogRetention(days int) {
	logRetentionDays.Store(int64(days))
}

// logCleanupWorker menghapus log lebih tua dari LOG_RETENTION_DAYS (jalankan sekali saat
// start + tiap jam; days <= 0 = nonaktif).
func logCleanupWorker() {
	run := func() {
		d := int(logRetentionDays.Load())
		if d <= 0 || UsersDB == nil {
			return
		}
		cutoff := time.Now().AddDate(0, 0, -d)
		res := UsersDB.Where("date < ?", cutoff).Delete(&models.TtpgLog{})
		if res.Error != nil {
			logger.Error("gagal hapus log lama: %v", res.Error)
			return
		}
		if res.RowsAffected > 0 {
			logger.Info("log cleanup: %d baris lebih tua dari %d hari dihapus", res.RowsAffected, d)
		}
	}
	time.Sleep(30 * time.Second)
	run()
	for range time.Tick(time.Hour) {
		run()
	}
}
