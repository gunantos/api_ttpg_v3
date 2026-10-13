package handler

import (
	"net/http"
	"strconv"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func ListLocations(c *gin.Context) {
	var data []models.PenginapanLocation
	if err := db.PenginapanDB.Order("id asc").Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil lokasi", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data lokasi ditemukan", data, nil)
}

func GetLocation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var row models.PenginapanLocation
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Lokasi tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Lokasi ditemukan", row, nil)
}

func CreateLocation(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Keterangan string `json:"keterangan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	row := models.PenginapanLocation{Name: req.Name, Keterangan: req.Keterangan}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat lokasi", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusCreated, "Lokasi berhasil dibuat", row, nil)
}

func UpdateLocation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		Name       *string `json:"name"`
		Keterangan *string `json:"keterangan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	var row models.PenginapanLocation
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Lokasi tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Keterangan != nil {
		updates["keterangan"] = *req.Keterangan
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update lokasi", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
	}
	response.Success(c, http.StatusOK, "Lokasi berhasil diupdate", row, nil)
}

func DeleteLocation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var used int64
	db.PenginapanDB.Model(&models.PenginapanRoom{}).Where("location_id = ?", id).Count(&used)
	if used > 0 {
		response.Error(c, http.StatusConflict, "Lokasi masih dipakai kamar", []response.ErrorItem{{Code: "IN_USE", Detail: "pindahkan kamar dulu"}})
		return
	}
	res := db.PenginapanDB.Delete(&models.PenginapanLocation{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus lokasi", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Lokasi tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Lokasi berhasil dihapus", nil, nil)
}

func ListRooms(c *gin.Context) {
	var data []models.PenginapanRoom
	q := db.PenginapanDB.Preload("Location").Order("id asc")
	if v := c.Query("location_id"); v != "" {
		q = q.Where("location_id = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if v := c.Query("room_type"); v != "" {
		q = q.Where("room_type = ?", v)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data kamar ditemukan", data, nil)
}

func GetRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var row models.PenginapanRoom
	if err := db.PenginapanDB.Preload("Location").First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Kamar tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Kamar ditemukan", row, nil)
}

func CreateRoom(c *gin.Context) {
	var req struct {
		LocationID *int   `json:"location_id"`
		RoomNumber string `json:"room_number" binding:"required"`
		RoomType   string `json:"room_type" binding:"required,oneof=pria wanita"`
		Capacity   *int   `json:"capacity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	row := models.PenginapanRoom{
		LocationID: req.LocationID,
		RoomNumber: req.RoomNumber,
		RoomType:   req.RoomType,
		Capacity:   1,
		Status:     "kosong",
	}
	if req.Capacity != nil && *req.Capacity > 0 {
		row.Capacity = *req.Capacity
	}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	broadcastRoomEvent("create", row.ID)
	response.Success(c, http.StatusCreated, "Kamar berhasil dibuat", row, nil)
}

func UpdateRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		LocationID *int    `json:"location_id"`
		RoomNumber *string `json:"room_number"`
		RoomType   *string `json:"room_type"`
		Capacity   *int    `json:"capacity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	if req.RoomType != nil && *req.RoomType != "pria" && *req.RoomType != "wanita" {
		response.Error(c, http.StatusBadRequest, "room_type harus pria/wanita", nil)
		return
	}
	var row models.PenginapanRoom
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Kamar tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.LocationID != nil {
		updates["location_id"] = *req.LocationID
	}
	if req.RoomNumber != nil {
		updates["room_number"] = *req.RoomNumber
	}
	if req.RoomType != nil {
		updates["room_type"] = *req.RoomType
	}
	if req.Capacity != nil {
		if *req.Capacity < row.CurrentOccupants {
			response.Error(c, http.StatusConflict, "capacity tidak boleh kurang dari penghuni sekarang", nil)
			return
		}
		updates["capacity"] = *req.Capacity
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		broadcastRoomEvent("update", row.ID)
	}
	response.Success(c, http.StatusOK, "Kamar berhasil diupdate", row, nil)
}

func UpdateRoomStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=kosong terisi maintenance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	var row models.PenginapanRoom
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Kamar tidak ditemukan", nil)
		return
	}
	if req.Status == "kosong" && row.CurrentOccupants > 0 {
		response.Error(c, http.StatusConflict, "kamar masih terisi, tidak bisa di-set kosong", nil)
		return
	}
	if req.Status == "terisi" {
		response.Error(c, http.StatusBadRequest, "status terisi dihitung otomatis dari check-in", nil)
		return
	}
	if err := db.PenginapanDB.Model(&row).Update("status", req.Status).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal update status kamar", nil)
		return
	}
	row.Status = req.Status
	broadcastRoomEvent("status", row.ID)
	response.Success(c, http.StatusOK, "Status kamar berhasil diupdate", row, nil)
}

func DeleteRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var active int64
	db.PenginapanDB.Model(&models.PenginapanGuest{}).Where("room_id = ? AND checkout_date IS NULL", id).Count(&active)
	if active > 0 {
		response.Error(c, http.StatusConflict, "Kamar masih ada penghuni aktif", nil)
		return
	}
	res := db.PenginapanDB.Delete(&models.PenginapanRoom{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus kamar", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Kamar tidak ditemukan", nil)
		return
	}
	broadcastRoomDeleted(id)
	response.Success(c, http.StatusOK, "Kamar berhasil dihapus", nil, nil)
}

func ListRoomLayouts(c *gin.Context) {
	var data []models.RoomLayout
	q := db.PenginapanDB.Preload("Location").Preload("Room").Order("urutan asc, id asc")
	if v := c.Query("location_id"); v != "" {
		q = q.Where("location_id = ?", v)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil layout kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data layout kamar ditemukan", data, nil)
}

func CreateRoomLayout(c *gin.Context) {
	var req struct {
		LocationID *int `json:"location_id"`
		RoomID     *int `json:"room_id"`
		RowNumber  *int `json:"row_number"`
		Urutan     int  `json:"urutan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	if req.RoomID != nil {
		var room models.PenginapanRoom
		if err := db.PenginapanDB.First(&room, *req.RoomID).Error; err != nil {
			response.Error(c, http.StatusBadRequest, "room_id tidak ditemukan", nil)
			return
		}
		if req.LocationID == nil {
			req.LocationID = room.LocationID
		}
	}
	row := models.RoomLayout{
		LocationID: req.LocationID,
		RoomID:     req.RoomID,
		RowNumber:  req.RowNumber,
		Urutan:     req.Urutan,
	}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat layout kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusCreated, "Layout kamar berhasil dibuat", row, nil)
}

func UpdateRoomLayout(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		LocationID *int `json:"location_id"`
		RoomID     *int `json:"room_id"`
		RowNumber  *int `json:"row_number"`
		Urutan     *int `json:"urutan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	var row models.RoomLayout
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Layout kamar tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.LocationID != nil {
		updates["location_id"] = *req.LocationID
	}
	if req.RoomID != nil {
		updates["room_id"] = *req.RoomID
	}
	if req.RowNumber != nil {
		updates["row_number"] = *req.RowNumber
	}
	if req.Urutan != nil {
		updates["urutan"] = *req.Urutan
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update layout kamar", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
	}
	response.Success(c, http.StatusOK, "Layout kamar berhasil diupdate", row, nil)
}

func DeleteRoomLayout(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	res := db.PenginapanDB.Delete(&models.RoomLayout{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus layout kamar", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Layout kamar tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Layout kamar berhasil dihapus", nil, nil)
}
