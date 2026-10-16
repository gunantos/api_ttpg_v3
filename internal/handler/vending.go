package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const vendingGlobalConfig = "__global__"
const vendingExtraMaxBytes = 8 * 1024

func vendingMerge(dst, src map[string]any) {
	for k, v := range src {
		if sm, ok := v.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				vendingMerge(dm, sm)
				continue
			}
		}
		dst[k] = v
	}
}

func vendingEnsureMachine(machineID string) {
	_, err := db.DB.Exec(`INSERT INTO vending_machines (machine_id, status) VALUES (?, 'pending') ON DUPLICATE KEY UPDATE id = id`, machineID)
	if err != nil {
		logger.Error("vending ensure machine: " + err.Error())
	}
}

func VendingConfig(c *gin.Context) {
	machineID := strings.TrimSpace(c.Query("machine_id"))
	if machineID == "" {
		response.Error(c, 400, "machine_id wajib diisi", nil)
		return
	}
	vendingEnsureMachine(machineID)

	var gPayload, gVersion string
	var gUpdated time.Time
	err := db.DB.QueryRow(`SELECT payload, version, updated_at FROM vending_configs WHERE machine_id = ?`, vendingGlobalConfig).
		Scan(&gPayload, &gVersion, &gUpdated)
	if err != nil {
		response.Error(c, 500, "config global tidak ditemukan — jalankan migration 006", nil)
		return
	}
	bundle := map[string]any{}
	if err := json.Unmarshal([]byte(gPayload), &bundle); err != nil {
		response.Error(c, 500, "payload config global tidak valid", nil)
		return
	}
	version, updated := gVersion, gUpdated

	var mPayload, mVersion string
	var mUpdated time.Time
	if err := db.DB.QueryRow(`SELECT payload, version, updated_at FROM vending_configs WHERE machine_id = ?`, machineID).
		Scan(&mPayload, &mVersion, &mUpdated); err == nil {
		override := map[string]any{}
		if err := json.Unmarshal([]byte(mPayload), &override); err != nil {
			logger.Warning("vending override tidak valid di " + machineID + ": " + err.Error())
		} else {
			vendingMerge(bundle, override)
			version, updated = mVersion, mUpdated
		}
	}

	db.DB.Exec(`UPDATE vending_machines SET last_seen_at = NOW(), config_version = ?, updated_at = NOW() WHERE machine_id = ?`, version, machineID)

	bundle["version"] = version
	bundle["updated_at"] = updated.Format(time.RFC3339)
	response.Success(c, 200, "Konfigurasi mesin", bundle, nil)
}

func VendingDeviceStatus(c *gin.Context) {
	var req struct {
		MachineID     string          `json:"machine_id"`
		AppVersion    string          `json:"app_version"`
		ConfigVersion string          `json:"config_version"`
		LastSyncAt    string          `json:"last_sync_at"`
		Extra         json.RawMessage `json:"extra"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	machineID := strings.TrimSpace(req.MachineID)
	if machineID == "" {
		response.Error(c, 400, "machine_id wajib diisi", nil)
		return
	}

	var lastSync any
	if strings.TrimSpace(req.LastSyncAt) != "" {
		t, err := time.Parse(time.RFC3339, req.LastSyncAt)
		if err != nil {
			response.Error(c, 400, "last_sync_at harus format RFC3339", nil)
			return
		}
		lastSync = t
	}

	var extra any
	if len(req.Extra) > 0 {
		if len(req.Extra) > vendingExtraMaxBytes {
			response.Error(c, 400, "extra maksimal 8KB", nil)
			return
		}
		obj := map[string]any{}
		if err := json.Unmarshal(req.Extra, &obj); err != nil {
			response.Error(c, 400, "extra harus objek JSON", nil)
			return
		}
		extra = string(req.Extra)
	}

	_, err := db.DB.Exec(`
		INSERT INTO vending_machines (machine_id, status, app_version, config_version, last_sync_at, last_seen_at, extra)
		VALUES (?, 'pending', ?, ?, ?, NOW(), ?)
		ON DUPLICATE KEY UPDATE
			app_version = VALUES(app_version),
			config_version = VALUES(config_version),
			last_sync_at = VALUES(last_sync_at),
			last_seen_at = NOW(),
			extra = VALUES(extra),
			updated_at = NOW()`,
		machineID,
		truncateStr(req.AppVersion, 30),
		truncateStr(req.ConfigVersion, 50),
		lastSync, extra)
	if err != nil {
		response.Error(c, 500, "gagal menyimpan status mesin: "+err.Error(), nil)
		return
	}

	var status string
	db.DB.QueryRow(`SELECT status FROM vending_machines WHERE machine_id = ?`, machineID).Scan(&status)
	response.Success(c, 200, "Heartbeat diterima", gin.H{"machine_id": machineID, "status": status}, nil)
}

func VendingMedia(c *gin.Context) {
	type mediaItem struct {
		ID       int    `json:"id"`
		Type     string `json:"type"`
		URL      string `json:"url"`
		Order    int    `json:"order"`
		Checksum string `json:"checksum"`
	}
	rows, err := db.DB.Query("SELECT id, type, url, `order`, COALESCE(checksum, '') FROM vending_media WHERE enabled = 1 ORDER BY `order`, id")
	if err != nil {
		response.Error(c, 500, "gagal membaca media: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	list := make([]mediaItem, 0)
	for rows.Next() {
		var m mediaItem
		if err := rows.Scan(&m.ID, &m.Type, &m.URL, &m.Order, &m.Checksum); err != nil {
			response.Error(c, 500, "gagal membaca media: "+err.Error(), nil)
			return
		}
		list = append(list, m)
	}
	response.Success(c, 200, "Daftar media attract", list, nil)
}

func truncateStr(s string, max int) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if len(s) > max {
		s = s[:max]
	}
	return s
}
