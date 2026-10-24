package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type vendingPlaylistReq struct {
	Name          string `json:"name"`
	Priority      int    `json:"priority"`
	LoopMode      string `json:"loop_mode"`
	RepeatCount   int    `json:"repeat_count"`
	WeightPercent int    `json:"weight_percent"`
	Days          []int  `json:"days"`
	DaysMask      *int   `json:"days_mask"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Enabled       *bool  `json:"enabled"`
}

type vendingPlaylistItemReq struct {
	MediaID         int   `json:"media_id" binding:"required"`
	SortOrder       int   `json:"sort_order"`
	RepeatCount     int   `json:"repeat_count"`
	WeightPercent   int   `json:"weight_percent"`
	DurationSeconds *int  `json:"duration_seconds"`
	Enabled         *bool `json:"enabled"`
}

type vendingMachinePlaylistReq struct {
	PlaylistID       int   `json:"playlist_id" binding:"required"`
	PriorityOverride *int  `json:"priority_override"`
	Enabled          *bool `json:"enabled"`
}

type vendingPlaybackEvent struct {
	PlaylistID      *int   `json:"playlist_id"`
	MediaID         *int   `json:"media_id"`
	StartedAt       string `json:"started_at" binding:"required"`
	DurationSeconds int    `json:"duration_seconds"`
	Completed       bool   `json:"completed"`
}

type vendingPlaybackReq struct {
	MachineID string                 `json:"machine_id" binding:"required"`
	Events    []vendingPlaybackEvent `json:"events" binding:"required"`
}

func vendingAdminOnly(c *gin.Context) bool {
	if c.GetString("role") != "admin" {
		response.Error(c, http.StatusForbidden, "Hanya admin yang boleh mengelola playlist vending", nil)
		return false
	}
	return true
}

func vendingDaysToMask(days []int) int {
	mask := 0
	for _, d := range days {
		if d >= 1 && d <= 7 {
			mask |= 1 << (d - 1)
		}
	}
	if mask == 0 {
		mask = 127
	}
	return mask
}

func vendingMaskToDays(mask int) []int {
	out := []int{}
	for i := 0; i < 7; i++ {
		if mask&(1<<i) != 0 {
			out = append(out, i+1)
		}
	}
	return out
}

func vendingNormalizeTime(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("format waktu harus HH:MM atau HH:MM:SS")
}

func vendingNullTime(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func vendingLoadPlaylists(machineID string) []map[string]any {
	ids := map[int]bool{}
	type assignment struct {
		playlistID  int
		priority    int
		hasOverride bool
		enabled     bool
	}
	assignments := []assignment{}
	rows, err := db.DB.Query(`
		SELECT vmp.playlist_id, COALESCE(vmp.priority_override, vp.priority), 
		       CASE WHEN vmp.priority_override IS NULL THEN 0 ELSE 1 END, vmp.enabled
		FROM vending_machine_playlists vmp
		JOIN vending_playlists vp ON vp.id = vmp.playlist_id
		WHERE vmp.machine_id IN (?, ?) AND vmp.enabled = 1 AND vp.enabled = 1`,
		"__global__", machineID)
	if err != nil {
		logger.Error("vending load assignments: " + err.Error())
		return []map[string]any{}
	}
	defer rows.Close()
	for rows.Next() {
		var a assignment
		var enabled int
		var hasOverride int
		if err := rows.Scan(&a.playlistID, &a.priority, &hasOverride, &enabled); err != nil {
			continue
		}
		a.hasOverride = hasOverride == 1
		a.enabled = enabled == 1
		if !a.enabled {
			continue
		}
		if _, ok := ids[a.playlistID]; ok {
			continue
		}
		ids[a.playlistID] = true
		assignments = append(assignments, a)
	}
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].priority != assignments[j].priority {
			return assignments[i].priority > assignments[j].priority
		}
		return assignments[i].playlistID < assignments[j].playlistID
	})

	out := []map[string]any{}
	for _, a := range assignments {
		var name, loopMode string
		var priority, repeatCount, weightPercent, daysMask int
		var startTime, endTime sql.NullString
		var enabled int
		err := db.DB.QueryRow(`
			SELECT name, priority, loop_mode, repeat_count, weight_percent, days_mask, start_time, end_time, enabled
			FROM vending_playlists WHERE id = ?`, a.playlistID).
			Scan(&name, &priority, &loopMode, &repeatCount, &weightPercent, &daysMask, &startTime, &endTime, &enabled)
		if err != nil || enabled != 1 {
			continue
		}
		if a.hasOverride {
			priority = a.priority
		}
		items := []map[string]any{}
		irows, err := db.DB.Query(`
			SELECT vpi.id, vpi.media_id, vpi.sort_order, vpi.repeat_count, vpi.weight_percent, vpi.duration_seconds, vpi.enabled,
			       vm.type, vm.url, COALESCE(vm.checksum,'')
			FROM vending_playlist_items vpi
			JOIN vending_media vm ON vm.id = vpi.media_id
			WHERE vpi.playlist_id = ? AND vpi.enabled = 1 AND vm.enabled = 1
			ORDER BY vpi.sort_order, vpi.id`, a.playlistID)
		if err == nil {
			defer irows.Close()
			for irows.Next() {
				var (
					itemID, mediaID, sortOrder, itemRepeat, itemWeight int
					dur                                                sql.NullInt64
					itemEnabled                                        int
					mediaType, mediaURL, checksum                      string
				)
				if err := irows.Scan(&itemID, &mediaID, &sortOrder, &itemRepeat, &itemWeight, &dur, &itemEnabled, &mediaType, &mediaURL, &checksum); err != nil {
					continue
				}
				if itemEnabled != 1 {
					continue
				}
				item := map[string]any{
					"id":             itemID,
					"media_id":       mediaID,
					"type":           mediaType,
					"url":            mediaURL,
					"checksum":       checksum,
					"sort_order":     sortOrder,
					"repeat_count":   itemRepeat,
					"weight_percent": itemWeight,
				}
				if dur.Valid {
					item["duration_seconds"] = int(dur.Int64)
				}
				items = append(items, item)
			}
		}
		pl := map[string]any{
			"id":             a.playlistID,
			"name":           name,
			"priority":       priority,
			"loop":           loopMode == "loop",
			"loop_mode":      loopMode,
			"repeat_count":   repeatCount,
			"weight_percent": weightPercent,
			"days":           vendingMaskToDays(daysMask),
			"days_mask":      daysMask,
			"items":          items,
		}
		if startTime.Valid {
			pl["start_time"] = startTime.String
		} else {
			pl["start_time"] = nil
		}
		if endTime.Valid {
			pl["end_time"] = endTime.String
		} else {
			pl["end_time"] = nil
		}
		out = append(out, pl)
	}
	return out
}

func VendingPlaylistsList(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	rows, err := db.DB.Query(`SELECT id, name, priority, loop_mode, repeat_count, weight_percent, days_mask, start_time, end_time, enabled, created_at, updated_at FROM vending_playlists ORDER BY priority DESC, id ASC`)
	if err != nil {
		response.Error(c, 500, "gagal membaca playlist: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var (
			id, priority, repeatCount, weightPercent, daysMask, enabled int
			name, loopMode                                              string
			startTime, endTime                                          sql.NullString
			createdAt, updatedAt                                        time.Time
		)
		if err := rows.Scan(&id, &name, &priority, &loopMode, &repeatCount, &weightPercent, &daysMask, &startTime, &endTime, &enabled, &createdAt, &updatedAt); err != nil {
			continue
		}
		item := map[string]any{
			"id":             id,
			"name":           name,
			"priority":       priority,
			"loop_mode":      loopMode,
			"loop":           loopMode == "loop",
			"repeat_count":   repeatCount,
			"weight_percent": weightPercent,
			"days":           vendingMaskToDays(daysMask),
			"days_mask":      daysMask,
			"enabled":        enabled == 1,
			"created_at":     createdAt.Format(time.RFC3339),
			"updated_at":     updatedAt.Format(time.RFC3339),
		}
		if startTime.Valid {
			item["start_time"] = startTime.String
		}
		if endTime.Valid {
			item["end_time"] = endTime.String
		}
		list = append(list, item)
	}
	response.Success(c, 200, "Daftar playlist vending", list, nil)
}

func VendingPlaylistCreate(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	var req vendingPlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.Error(c, 400, "name wajib diisi", nil)
		return
	}
	if req.LoopMode == "" {
		req.LoopMode = "loop"
	}
	if req.LoopMode != "loop" && req.LoopMode != "once" {
		response.Error(c, 400, "loop_mode harus loop atau once", nil)
		return
	}
	if req.RepeatCount <= 0 {
		req.RepeatCount = 1
	}
	if req.WeightPercent <= 0 || req.WeightPercent > 100 {
		req.WeightPercent = 100
	}
	mask := 127
	if req.DaysMask != nil {
		mask = *req.DaysMask
	} else if len(req.Days) > 0 {
		mask = vendingDaysToMask(req.Days)
	}
	start, err := vendingNormalizeTime(req.StartTime)
	if err != nil {
		response.Error(c, 400, err.Error(), nil)
		return
	}
	end, err := vendingNormalizeTime(req.EndTime)
	if err != nil {
		response.Error(c, 400, err.Error(), nil)
		return
	}
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	res, err := db.DB.Exec(`INSERT INTO vending_playlists (name, priority, loop_mode, repeat_count, weight_percent, days_mask, start_time, end_time, enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(req.Name), req.Priority, req.LoopMode, req.RepeatCount, req.WeightPercent, mask, vendingNullTime(start), vendingNullTime(end), enabled)
	if err != nil {
		response.Error(c, 500, "gagal membuat playlist: "+err.Error(), nil)
		return
	}
	id, _ := res.LastInsertId()
	response.Success(c, 201, "Playlist berhasil dibuat", gin.H{"id": id}, nil)
}

func VendingPlaylistGet(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	var (
		name, loopMode                                          string
		priority, repeatCount, weightPercent, daysMask, enabled int
		startTime, endTime                                      sql.NullString
		createdAt, updatedAt                                    time.Time
	)
	err = db.DB.QueryRow(`SELECT name, priority, loop_mode, repeat_count, weight_percent, days_mask, start_time, end_time, enabled, created_at, updated_at FROM vending_playlists WHERE id = ?`, id).
		Scan(&name, &priority, &loopMode, &repeatCount, &weightPercent, &daysMask, &startTime, &endTime, &enabled, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		response.Error(c, 404, "playlist tidak ditemukan", nil)
		return
	}
	if err != nil {
		response.Error(c, 500, "gagal membaca playlist: "+err.Error(), nil)
		return
	}
	item := map[string]any{
		"id":             id,
		"name":           name,
		"priority":       priority,
		"loop_mode":      loopMode,
		"loop":           loopMode == "loop",
		"repeat_count":   repeatCount,
		"weight_percent": weightPercent,
		"days":           vendingMaskToDays(daysMask),
		"days_mask":      daysMask,
		"enabled":        enabled == 1,
		"created_at":     createdAt.Format(time.RFC3339),
		"updated_at":     updatedAt.Format(time.RFC3339),
	}
	if startTime.Valid {
		item["start_time"] = startTime.String
	}
	if endTime.Valid {
		item["end_time"] = endTime.String
	}
	response.Success(c, 200, "Detail playlist", item, nil)
}

func VendingPlaylistUpdate(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	var req vendingPlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	sets := []string{}
	args := []any{}
	if strings.TrimSpace(req.Name) != "" {
		sets = append(sets, "name = ?")
		args = append(args, strings.TrimSpace(req.Name))
	}
	if req.Priority != 0 {
		sets = append(sets, "priority = ?")
		args = append(args, req.Priority)
	}
	if req.LoopMode != "" {
		if req.LoopMode != "loop" && req.LoopMode != "once" {
			response.Error(c, 400, "loop_mode harus loop atau once", nil)
			return
		}
		sets = append(sets, "loop_mode = ?")
		args = append(args, req.LoopMode)
	}
	if req.RepeatCount > 0 {
		sets = append(sets, "repeat_count = ?")
		args = append(args, req.RepeatCount)
	}
	if req.WeightPercent > 0 && req.WeightPercent <= 100 {
		sets = append(sets, "weight_percent = ?")
		args = append(args, req.WeightPercent)
	}
	if req.DaysMask != nil || len(req.Days) > 0 {
		mask := 127
		if req.DaysMask != nil {
			mask = *req.DaysMask
		} else {
			mask = vendingDaysToMask(req.Days)
		}
		sets = append(sets, "days_mask = ?")
		args = append(args, mask)
	}
	if req.StartTime != "" {
		start, err := vendingNormalizeTime(req.StartTime)
		if err != nil {
			response.Error(c, 400, err.Error(), nil)
			return
		}
		sets = append(sets, "start_time = ?")
		args = append(args, vendingNullTime(start))
	}
	if req.EndTime != "" {
		end, err := vendingNormalizeTime(req.EndTime)
		if err != nil {
			response.Error(c, 400, err.Error(), nil)
			return
		}
		sets = append(sets, "end_time = ?")
		args = append(args, vendingNullTime(end))
	}
	if req.Enabled != nil {
		enabled := 0
		if *req.Enabled {
			enabled = 1
		}
		sets = append(sets, "enabled = ?")
		args = append(args, enabled)
	}
	if len(sets) == 0 {
		response.Error(c, 400, "tidak ada field yang diupdate", nil)
		return
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	q := "UPDATE vending_playlists SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := db.DB.Exec(q, args...); err != nil {
		response.Error(c, 500, "gagal update playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Playlist berhasil diupdate", gin.H{"id": id}, nil)
}

func VendingPlaylistDelete(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	if _, err := db.DB.Exec(`DELETE FROM vending_playlists WHERE id = ?`, id); err != nil {
		response.Error(c, 500, "gagal hapus playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Playlist berhasil dihapus", gin.H{"id": id}, nil)
}

func VendingPlaylistItemsList(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	playlistID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id playlist tidak valid", nil)
		return
	}
	rows, err := db.DB.Query(`
		SELECT vpi.id, vpi.media_id, vpi.sort_order, vpi.repeat_count, vpi.weight_percent, vpi.duration_seconds, vpi.enabled,
		       vm.type, vm.url, COALESCE(vm.checksum,'')
		FROM vending_playlist_items vpi
		JOIN vending_media vm ON vm.id = vpi.media_id
		WHERE vpi.playlist_id = ?
		ORDER BY vpi.sort_order, vpi.id`, playlistID)
	if err != nil {
		response.Error(c, 500, "gagal membaca item playlist: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var (
			id, mediaID, sortOrder, repeatCount, weightPercent, enabled int
			dur                                                         sql.NullInt64
			mediaType, mediaURL, checksum                               string
		)
		if err := rows.Scan(&id, &mediaID, &sortOrder, &repeatCount, &weightPercent, &dur, &enabled, &mediaType, &mediaURL, &checksum); err != nil {
			continue
		}
		item := map[string]any{
			"id":             id,
			"media_id":       mediaID,
			"media_type":     mediaType,
			"url":            mediaURL,
			"checksum":       checksum,
			"sort_order":     sortOrder,
			"repeat_count":   repeatCount,
			"weight_percent": weightPercent,
			"enabled":        enabled == 1,
		}
		if dur.Valid {
			item["duration_seconds"] = int(dur.Int64)
		}
		list = append(list, item)
	}
	response.Success(c, 200, "Daftar item playlist", list, nil)
}

func VendingPlaylistItemCreate(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	playlistID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id playlist tidak valid", nil)
		return
	}
	var req vendingPlaylistItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	if req.RepeatCount <= 0 {
		req.RepeatCount = 1
	}
	if req.WeightPercent <= 0 || req.WeightPercent > 100 {
		req.WeightPercent = 100
	}
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	var dur any
	if req.DurationSeconds != nil {
		dur = *req.DurationSeconds
	}
	res, err := db.DB.Exec(`INSERT INTO vending_playlist_items (playlist_id, media_id, sort_order, repeat_count, weight_percent, duration_seconds, enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		playlistID, req.MediaID, req.SortOrder, req.RepeatCount, req.WeightPercent, dur, enabled)
	if err != nil {
		response.Error(c, 500, "gagal membuat item playlist: "+err.Error(), nil)
		return
	}
	id, _ := res.LastInsertId()
	response.Success(c, 201, "Item playlist berhasil dibuat", gin.H{"id": id}, nil)
}

func VendingPlaylistItemUpdate(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	var req vendingPlaylistItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	sets := []string{}
	args := []any{}
	if req.MediaID > 0 {
		sets = append(sets, "media_id = ?")
		args = append(args, req.MediaID)
	}
	sets = append(sets, "sort_order = ?")
	args = append(args, req.SortOrder)
	if req.RepeatCount > 0 {
		sets = append(sets, "repeat_count = ?")
		args = append(args, req.RepeatCount)
	}
	if req.WeightPercent > 0 && req.WeightPercent <= 100 {
		sets = append(sets, "weight_percent = ?")
		args = append(args, req.WeightPercent)
	}
	if req.DurationSeconds != nil {
		sets = append(sets, "duration_seconds = ?")
		args = append(args, *req.DurationSeconds)
	}
	if req.Enabled != nil {
		enabled := 0
		if *req.Enabled {
			enabled = 1
		}
		sets = append(sets, "enabled = ?")
		args = append(args, enabled)
	}
	if len(sets) == 0 {
		response.Error(c, 400, "tidak ada field yang diupdate", nil)
		return
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	q := "UPDATE vending_playlist_items SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := db.DB.Exec(q, args...); err != nil {
		response.Error(c, 500, "gagal update item playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Item playlist berhasil diupdate", gin.H{"id": id}, nil)
}

func VendingPlaylistItemDelete(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, 400, "id tidak valid", nil)
		return
	}
	if _, err := db.DB.Exec(`DELETE FROM vending_playlist_items WHERE id = ?`, id); err != nil {
		response.Error(c, 500, "gagal hapus item playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Item playlist berhasil dihapus", gin.H{"id": id}, nil)
}

func VendingMachinePlaylistsList(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	machineID := strings.TrimSpace(c.Param("id"))
	rows, err := db.DB.Query(`
		SELECT vmp.id, vmp.machine_id, vmp.playlist_id, vp.name, vmp.priority_override, vmp.enabled, vmp.created_at, vmp.updated_at
		FROM vending_machine_playlists vmp
		JOIN vending_playlists vp ON vp.id = vmp.playlist_id
		WHERE vmp.machine_id = ?
		ORDER BY COALESCE(vmp.priority_override, vp.priority) DESC, vmp.id ASC`, machineID)
	if err != nil {
		response.Error(c, 500, "gagal membaca assignment playlist: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var (
			id, playlistID, enabled int
			mID, name               string
			prioOverride            sql.NullInt64
			createdAt, updatedAt    time.Time
		)
		if err := rows.Scan(&id, &mID, &playlistID, &name, &prioOverride, &enabled, &createdAt, &updatedAt); err != nil {
			continue
		}
		item := map[string]any{
			"id":          id,
			"machine_id":  mID,
			"playlist_id": playlistID,
			"name":        name,
			"enabled":     enabled == 1,
			"created_at":  createdAt.Format(time.RFC3339),
			"updated_at":  updatedAt.Format(time.RFC3339),
		}
		if prioOverride.Valid {
			item["priority_override"] = int(prioOverride.Int64)
		}
		list = append(list, item)
	}
	response.Success(c, 200, "Daftar assignment playlist mesin", list, nil)
}

func VendingMachinePlaylistAssign(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	machineID := strings.TrimSpace(c.Param("id"))
	var req vendingMachinePlaylistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	var prio any
	if req.PriorityOverride != nil {
		prio = *req.PriorityOverride
	}
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	_, err := db.DB.Exec(`
		INSERT INTO vending_machine_playlists (machine_id, playlist_id, priority_override, enabled)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE priority_override = VALUES(priority_override), enabled = VALUES(enabled), updated_at = NOW()`,
		machineID, req.PlaylistID, prio, enabled)
	if err != nil {
		response.Error(c, 500, "gagal assign playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Playlist berhasil di-assign ke mesin", gin.H{"machine_id": machineID, "playlist_id": req.PlaylistID}, nil)
}

func VendingMachinePlaylistUnassign(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	machineID := strings.TrimSpace(c.Param("id"))
	playlistID, err := strconv.Atoi(c.Param("playlist_id"))
	if err != nil {
		response.Error(c, 400, "playlist_id tidak valid", nil)
		return
	}
	if _, err := db.DB.Exec(`DELETE FROM vending_machine_playlists WHERE machine_id = ? AND playlist_id = ?`, machineID, playlistID); err != nil {
		response.Error(c, 500, "gagal unassign playlist: "+err.Error(), nil)
		return
	}
	response.Success(c, 200, "Assignment playlist berhasil dihapus", gin.H{"machine_id": machineID, "playlist_id": playlistID}, nil)
}

func VendingPlayback(c *gin.Context) {
	var req vendingPlaybackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "body tidak valid", nil)
		return
	}
	machineID := strings.TrimSpace(req.MachineID)
	if machineID == "" {
		response.Error(c, 400, "machine_id wajib diisi", nil)
		return
	}
	if len(req.Events) == 0 || len(req.Events) > 100 {
		response.Error(c, 400, "events harus 1-100 item", nil)
		return
	}
	db.DB.Exec(`INSERT INTO vending_machines (machine_id, status) VALUES (?, 'pending') ON DUPLICATE KEY UPDATE id = id`, machineID)

	for _, ev := range req.Events {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(ev.StartedAt))
		if err != nil {
			response.Error(c, 400, "started_at harus format RFC3339", nil)
			return
		}
		completed := 0
		if ev.Completed {
			completed = 1
		}
		if _, err := db.DB.Exec(`INSERT INTO vending_play_logs (machine_id, playlist_id, media_id, started_at, duration_seconds, completed) VALUES (?, ?, ?, ?, ?, ?)`,
			machineID, ev.PlaylistID, ev.MediaID, t, ev.DurationSeconds, completed); err != nil {
			response.Error(c, 500, "gagal menyimpan log playback: "+err.Error(), nil)
			return
		}
	}
	db.DB.Exec(`UPDATE vending_machines SET last_seen_at = NOW(), updated_at = NOW() WHERE machine_id = ?`, machineID)
	response.Success(c, 200, "Log playback berhasil disimpan", gin.H{"machine_id": machineID, "count": len(req.Events)}, nil)
}

func VendingPlaybackReport(c *gin.Context) {
	if !vendingAdminOnly(c) {
		return
	}
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" {
		from = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	where := "started_at >= ? AND started_at < ?"
	args := []any{from + " 00:00:00", to + " 23:59:59"}
	if v := strings.TrimSpace(c.Query("machine_id")); v != "" {
		where += " AND machine_id = ?"
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("playlist_id")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			where += " AND playlist_id = ?"
			args = append(args, n)
		}
	}
	if v := strings.TrimSpace(c.Query("media_id")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			where += " AND media_id = ?"
			args = append(args, n)
		}
	}
	groupBy := strings.ToLower(strings.TrimSpace(c.Query("group_by")))
	if groupBy == "" {
		groupBy = "day,media"
	}
	// total for percentage
	var total int
	db.DB.QueryRow("SELECT COUNT(*) FROM vending_play_logs WHERE "+where, args...).Scan(&total)

	groupExpr := "DATE(started_at)"
	switch groupBy {
	case "hour":
		groupExpr = "DATE(started_at), HOUR(started_at)"
	case "machine":
		groupExpr = "machine_id"
	case "media":
		groupExpr = "media_id"
	case "playlist":
		groupExpr = "playlist_id"
	case "day,media":
		groupExpr = "DATE(started_at), media_id"
	case "day,hour":
		groupExpr = "DATE(started_at), HOUR(started_at)"
	}

	q := "SELECT " + groupExpr + ", COUNT(*) AS play_count, COALESCE(SUM(duration_seconds),0) AS total_duration, COALESCE(SUM(completed),0) AS completed_count FROM vending_play_logs WHERE " + where + " GROUP BY " + groupExpr + " ORDER BY play_count DESC"
	rows, err := db.DB.Query(q, args...)
	if err != nil {
		response.Error(c, 500, "gagal membaca laporan playback: "+err.Error(), nil)
		return
	}
	defer rows.Close()
	// Re-run with columns scan:
	rows2, err := db.DB.Query(q, args...)
	if err != nil {
		response.Error(c, 500, "gagal membaca laporan playback: "+err.Error(), nil)
		return
	}
	defer rows2.Close()
	cols, _ := rows2.Columns()
	list := []map[string]any{}
	for rows2.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows2.Scan(ptrs...); err != nil {
			continue
		}
		row := map[string]any{}
		for i, col := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		pc, _ := row["play_count"].(int64)
		if total > 0 {
			row["percentage"] = float64(pc) / float64(total) * 100
		} else {
			row["percentage"] = 0.0
		}
		list = append(list, row)
	}
	response.Success(c, 200, "Laporan playback vending", gin.H{"total": total, "from": from, "to": to, "rows": list}, nil)
}
