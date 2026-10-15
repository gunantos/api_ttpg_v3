package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// ListActivityLogs — log request milik user (JWT); admin melihat semua (+ filter user_id).
func ListActivityLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}

	q := db.UsersDB.Model(&models.TtpgLog{})

	role, _ := c.Get("role")
	isAdmin := role == "admin"
	if !isAdmin {
		uid := 0
		if v, ok := c.Get("user_id"); ok {
			if id, ok2 := v.(uint); ok2 {
				uid = int(id)
			}
		}
		q = q.Where("id_user = ?", uid)
	} else if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		if uid, err := strconv.Atoi(v); err == nil {
			q = q.Where("id_user = ?", uid)
		}
	}
	if v := strings.TrimSpace(c.Query("method")); v != "" {
		q = q.Where("method = ?", strings.ToUpper(v))
	}
	if v := strings.TrimSpace(c.Query("code")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q = q.Where("response_code = ?", n)
		}
	}
	if v := strings.TrimSpace(c.Query("path")); v != "" {
		q = q.Where("uri LIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(c.Query("authorized")); v == "y" || v == "n" {
		q = q.Where("authorized = ?", v)
	}
	if v := strings.TrimSpace(c.Query("from")); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("date >= ?", t)
		}
	}
	if v := strings.TrimSpace(c.Query("to")); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("date < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil log", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	var data []models.TtpgLog
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil log", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	page := offset/limit + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	response.Success(c, http.StatusOK, "Log aktivitas ditemukan", data, &response.Pagination{
		Page: page, Limit: limit, Total: int(total), TotalPages: totalPages,
	})
}
