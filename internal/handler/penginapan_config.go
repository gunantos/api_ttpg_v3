package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func ListConfigs(c *gin.Context) {
	var data []models.PenginapanConfig
	q := db.PenginapanDB.Order("id_configs asc")
	if v := strings.TrimSpace(c.Query("key")); v != "" {
		q = q.Where("key_configs = ?", v)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil configs", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data configs ditemukan", data, nil)
}

func GetConfigByKey(c *gin.Context) {
	var data models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", c.Param("key")).First(&data).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Config tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Config ditemukan", data, nil)
}

func CreateConfig(c *gin.Context) {
	var req struct {
		KeyConfigs   string  `json:"key_configs" binding:"required"`
		ValueConfigs *string `json:"value_configs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	row := models.PenginapanConfig{KeyConfigs: req.KeyConfigs, ValueConfigs: req.ValueConfigs}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat config", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusCreated, "Config berhasil dibuat", row, nil)
}

func UpdateConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		KeyConfigs   *string `json:"key_configs"`
		ValueConfigs *string `json:"value_configs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	var row models.PenginapanConfig
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Config tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.KeyConfigs != nil {
		updates["key_configs"] = *req.KeyConfigs
	}
	if req.ValueConfigs != nil {
		updates["value_configs"] = *req.ValueConfigs
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update config", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
	}
	response.Success(c, http.StatusOK, "Config berhasil diupdate", row, nil)
}

func DeleteConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	res := db.PenginapanDB.Delete(&models.PenginapanConfig{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus config", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Config tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Config berhasil dihapus", nil, nil)
}

func ListPaymentMethods(c *gin.Context) {
	var data []models.PaymentMethod
	q := db.PenginapanDB.Order("sort_order asc, id asc")
	if c.Query("active_only") == "true" {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil metode pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data metode pembayaran ditemukan", data, nil)
}

func CreatePaymentMethod(c *gin.Context) {
	var req struct {
		Code      string `json:"code" binding:"required"`
		Name      string `json:"name" binding:"required"`
		IsActive  *bool  `json:"is_active"`
		SortOrder *int   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	row := models.PaymentMethod{Code: req.Code, Name: req.Name}
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	} else {
		row.IsActive = true
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat metode pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusCreated, "Metode pembayaran berhasil dibuat", row, nil)
}

func UpdatePaymentMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		Code      *string `json:"code"`
		Name      *string `json:"name"`
		IsActive  *bool   `json:"is_active"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	var row models.PaymentMethod
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Metode pembayaran tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.Code != nil {
		updates["code"] = *req.Code
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update metode pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
	}
	response.Success(c, http.StatusOK, "Metode pembayaran berhasil diupdate", row, nil)
}

func DeletePaymentMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var used int64
	db.PenginapanDB.Model(&models.Payment{}).Where("payment_method_id = ?", id).Count(&used)
	if used > 0 {
		response.Error(c, http.StatusConflict, "Metode pembayaran masih dipakai pembayaran", []response.ErrorItem{{Code: "IN_USE", Detail: "nonaktifkan saja (is_active=false)"}})
		return
	}
	res := db.PenginapanDB.Delete(&models.PaymentMethod{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus metode pembayaran", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Metode pembayaran tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Metode pembayaran berhasil dihapus", nil, nil)
}

func ListPurchaseSources(c *gin.Context) {
	var data []models.PurchaseSource
	q := db.PenginapanDB.Order("sort_order asc, id asc")
	if c.Query("active_only") == "true" {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil sumber pembelian", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data sumber pembelian ditemukan", data, nil)
}

func CreatePurchaseSource(c *gin.Context) {
	var req struct {
		Code      string `json:"code" binding:"required"`
		Name      string `json:"name" binding:"required"`
		IsActive  *bool  `json:"is_active"`
		SortOrder *int   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}
	row := models.PurchaseSource{Code: req.Code, Name: req.Name}
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	} else {
		row.IsActive = true
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := db.PenginapanDB.Create(&row).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membuat sumber pembelian", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusCreated, "Sumber pembelian berhasil dibuat", row, nil)
}

func UpdatePurchaseSource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		Code      *string `json:"code"`
		Name      *string `json:"name"`
		IsActive  *bool   `json:"is_active"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	var row models.PurchaseSource
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Sumber pembelian tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.Code != nil {
		updates["code"] = *req.Code
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if len(updates) > 0 {
		if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update sumber pembelian", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
	}
	response.Success(c, http.StatusOK, "Sumber pembelian berhasil diupdate", row, nil)
}

func DeletePurchaseSource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var used int64
	db.PenginapanDB.Model(&models.Payment{}).Where("source_id = ?", id).Count(&used)
	if used > 0 {
		response.Error(c, http.StatusConflict, "Sumber pembelian masih dipakai pembayaran", []response.ErrorItem{{Code: "IN_USE", Detail: "nonaktifkan saja (is_active=false)"}})
		return
	}
	res := db.PenginapanDB.Delete(&models.PurchaseSource{}, id)
	if res.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal hapus sumber pembelian", nil)
		return
	}
	if res.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Sumber pembelian tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Sumber pembelian berhasil dihapus", nil, nil)
}

func GetQRISPenginapan(c *gin.Context) {
	var payloadCfg models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", "qris_payload").First(&payloadCfg).Error; err != nil ||
		payloadCfg.ValueConfigs == nil || strings.TrimSpace(*payloadCfg.ValueConfigs) == "" {
		response.Error(c, http.StatusNotFound, "QRIS belum dikonfigurasi", []response.ErrorItem{{
			Code:   "NOT_CONFIGURED",
			Detail: "isi key_configs 'qris_payload' (payload EMV QR / string QRIS statis) via POST /penginapan/configs",
		}})
		return
	}

	data := gin.H{"payload": strings.TrimSpace(*payloadCfg.ValueConfigs)}
	var merchantCfg models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", "qris_merchant").First(&merchantCfg).Error; err == nil &&
		merchantCfg.ValueConfigs != nil {
		data["merchant"] = strings.TrimSpace(*merchantCfg.ValueConfigs)
	}
	response.Success(c, http.StatusOK, "Konfigurasi QRIS ditemukan", data, nil)
}
