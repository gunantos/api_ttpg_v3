package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/andtho89/ttpg/internal/request"
	"github.com/gin-gonic/gin"
)

func ListPayments(c *gin.Context) {
	var data []models.Payment
	q := db.PenginapanDB.
		Preload("PaymentMethod").
		Preload("Source").
		Preload("Guests.Guest").
		Order("id desc").
		Limit(500)
	if v := c.Query("guest_id"); v != "" {
		q = q.Where("id IN (?)", db.PenginapanDB.Model(&models.PaymentGuest{}).Select("payment_id").Where("guest_id = ?", v))
	}
	if v := c.Query("source_id"); v != "" {
		q = q.Where("source_id = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if v := c.Query("payment_method_id"); v != "" {
		q = q.Where("payment_method_id = ?", v)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data pembayaran ditemukan", data, nil)
}

func GetPayment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var row models.Payment
	if err := db.PenginapanDB.
		Preload("PaymentMethod").
		Preload("Source").
		Preload("Guests.Guest").
		First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}
	response.Success(c, http.StatusOK, "Pembayaran ditemukan", row, nil)
}

func CreatePaymentPenginapan(c *gin.Context) {
	var req request.PenginapanCreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}

	allocations, err := buildManualAllocations(req.Allocations, req.Amount, req.GuestIDs)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	now := time.Now()
	tx := db.PenginapanDB.Begin()
	if tx.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	defer tx.Rollback()

	for _, gid := range req.GuestIDs {
		var guest models.PenginapanGuest
		if err := tx.First(&guest, gid).Error; err != nil {
			response.Error(c, http.StatusNotFound, fmt.Sprintf("guest_id %d tidak ditemukan", gid), nil)
			return
		}
	}

	payment, err := createPaymentTx(tx, req.Amount, req.PaymentMethodID, req.SourceID, req.Status, req.Note, penginapanUserID(c), now)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	for _, a := range allocations {
		pg := models.PaymentGuest{PaymentID: payment.ID, GuestID: a.GuestID, Amount: a.Amount}
		if err := tx.Create(&pg).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menyimpan alokasi pembayaran", nil)
			return
		}
		payment.Guests = append(payment.Guests, pg)
	}

	if err := tx.Commit().Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal commit transaksi", nil)
		return
	}

	response.Success(c, http.StatusCreated, "Pembayaran berhasil dibuat", payment, nil)
}

func buildManualAllocations(allocs []request.PenginapanAllocation, amount int, guestIDs []int) ([]request.PenginapanAllocation, error) {
	if len(allocs) == 0 {
		if len(guestIDs) == 1 {
			return []request.PenginapanAllocation{{GuestID: guestIDs[0], Amount: amount}}, nil
		}
		return nil, errors.New("allocations wajib diisi untuk lebih dari 1 tamu")
	}
	total := 0
	seen := map[int]bool{}
	for _, a := range allocs {
		if a.GuestID == 0 {
			return nil, errors.New("allocations: guest_id wajib diisi")
		}
		if seen[a.GuestID] {
			return nil, fmt.Errorf("allocations: guest_id %d duplikat", a.GuestID)
		}
		seen[a.GuestID] = true
		if a.Amount < 0 {
			return nil, errors.New("allocations: amount tidak boleh negatif")
		}
		found := false
		for _, gid := range guestIDs {
			if gid == a.GuestID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("allocations: guest_id %d tidak ada di guest_ids", a.GuestID)
		}
		total += a.Amount
	}
	if total != amount {
		return nil, fmt.Errorf("total allocations (%d) harus sama dengan amount pembayaran (%d)", total, amount)
	}
	return allocs, nil
}

func UpdatePaymentPenginapan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req request.PenginapanUpdatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	if req.Status != nil && *req.Status != "pending" && *req.Status != "lunas" {
		response.Error(c, http.StatusBadRequest, "status harus pending atau lunas (gunakan DELETE untuk batal)", nil)
		return
	}
	var row models.Payment
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}
	if row.Status == "batal" {
		response.Error(c, http.StatusConflict, "pembayaran batal tidak bisa diubah", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.Amount != nil {
		var total int
		db.PenginapanDB.Model(&models.PaymentGuest{}).Where("payment_id = ?", id).Select("COALESCE(SUM(amount),0)").Scan(&total)
		if *req.Amount < total {
			response.Error(c, http.StatusConflict, fmt.Sprintf("amount tidak boleh kurang dari total alokasi (%d)", total), nil)
			return
		}
		updates["amount"] = *req.Amount
	}
	if req.PaymentMethodID != nil {
		var method models.PaymentMethod
		if err := db.PenginapanDB.First(&method, *req.PaymentMethodID).Error; err != nil {
			response.Error(c, http.StatusBadRequest, "payment_method_id tidak ditemukan", nil)
			return
		}
		updates["payment_method_id"] = *req.PaymentMethodID
	}
	if req.SourceID != nil {
		var source models.PurchaseSource
		if err := db.PenginapanDB.First(&source, *req.SourceID).Error; err != nil {
			response.Error(c, http.StatusBadRequest, "source_id tidak ditemukan", nil)
			return
		}
		updates["source_id"] = *req.SourceID
	}
	if req.Note != nil {
		updates["note"] = *req.Note
	}
	if req.Status != nil {
		updates["status"] = *req.Status
		if *req.Status == "lunas" && row.PaidAt == nil {
			now := time.Now()
			updates["paid_at"] = now
		}
	}
	if len(updates) == 0 {
		response.Error(c, http.StatusBadRequest, "tidak ada field untuk diupdate", nil)
		return
	}
	if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal update pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	db.PenginapanDB.Preload("PaymentMethod").Preload("Source").Preload("Guests.Guest").First(&row, id)
	response.Success(c, http.StatusOK, "Pembayaran berhasil diupdate", row, nil)
}

func VoidPaymentPenginapan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var row models.Payment
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}
	if row.Status == "batal" {
		response.Error(c, http.StatusConflict, "pembayaran sudah batal", nil)
		return
	}
	if err := db.PenginapanDB.Model(&row).Update("status", "batal").Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal membatalkan pembayaran", nil)
		return
	}
	response.Success(c, http.StatusOK, "Pembayaran berhasil dibatalkan", nil, nil)
}
