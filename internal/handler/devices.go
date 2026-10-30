package handler

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type MonitorDeviceRow struct {
	ID            int        `json:"id"`
	Type          string     `json:"type"`
	DeviceID      string     `json:"device_id"`
	Name          string     `json:"name"`
	Location      string     `json:"location"`
	Status        string     `json:"status"`
	AppVersion    string     `json:"app_version"`
	ConfigVersion string     `json:"config_version"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	IPAddress     string     `json:"ip_address"`
	Resolution    string     `json:"resolution"`
	Volume        int        `json:"volume"`
	Brightness    int        `json:"brightness"`
}

func ListMonitorDevices(c *gin.Context) {
	limit := 20
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := strings.TrimSpace(c.Query("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	where := "1=1"
	args := []any{}
	switch t := strings.ToLower(strings.TrimSpace(c.Query("type"))); t {
	case "":
	case "vending", "signage":
		where += " AND type = ?"
		args = append(args, t)
	default:
		response.Error(c, 400, "type harus vending/signage", nil)
		return
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		where += " AND (device_id LIKE ? OR name LIKE ? OR location LIKE ?)"
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		where += " AND status = ?"
		args = append(args, s)
	}

	var total int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE "+where, args...).Scan(&total); err != nil {
		response.Error(c, 500, "gagal menghitung device: "+err.Error(), nil)
		return
	}
	rows, err := db.DB.Query(`SELECT id, type, device_id, COALESCE(name,''), COALESCE(location,''), COALESCE(status,'pending'),
		COALESCE(app_version,''), COALESCE(config_version,''), last_seen_at, last_sync_at, COALESCE(ip_address,''),
		COALESCE(resolution,''), COALESCE(volume,80), COALESCE(brightness,80)
		FROM devices WHERE `+where+` ORDER BY last_seen_at DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		response.Error(c, 500, "gagal membaca device: "+err.Error(), nil)
		return
	}
	defer rows.Close()

	list := make([]MonitorDeviceRow, 0)
	for rows.Next() {
		var m MonitorDeviceRow
		var lastSeen, lastSync sql.NullTime
		if err := rows.Scan(&m.ID, &m.Type, &m.DeviceID, &m.Name, &m.Location, &m.Status,
			&m.AppVersion, &m.ConfigVersion, &lastSeen, &lastSync, &m.IPAddress,
			&m.Resolution, &m.Volume, &m.Brightness); err != nil {
			response.Error(c, 500, "gagal membaca device: "+err.Error(), nil)
			return
		}
		if lastSeen.Valid {
			m.LastSeenAt = &lastSeen.Time
		}
		if lastSync.Valid {
			m.LastSyncAt = &lastSync.Time
		}
		list = append(list, m)
	}

	page := offset/limit + 1
	totalPages := (total + limit - 1) / limit
	response.Success(c, 200, "Daftar device", list, &response.Pagination{
		Page: page, Limit: limit, Total: total, TotalPages: totalPages,
	})
}
