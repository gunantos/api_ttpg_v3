package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/andtho89/ttpg/internal/request"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func penginapanUserID(c *gin.Context) *uint {
	v, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	id, ok := v.(uint)
	if !ok {
		return nil
	}
	return &id
}

func parsePenginapanDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("format tanggal harus YYYY-MM-DD")
	}
	return &t, nil
}

func parsePenginapanTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("format jam harus HH:MM atau HH:MM:SS")
}

func attachGuestPaymentSummary(guests []models.PenginapanGuest) error {
	if len(guests) == 0 {
		return nil
	}
	ids := make([]int, 0, len(guests))
	for _, g := range guests {
		ids = append(ids, g.ID)
	}
	rows, err := db.PenginapanDB.Raw(`
		SELECT pg.guest_id, COALESCE(SUM(pg.amount), 0)
		FROM payment_guests pg
		JOIN payments p ON p.id = pg.payment_id
		WHERE p.status = 'lunas' AND pg.guest_id IN ?
		GROUP BY pg.guest_id
	`, ids).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	totals := map[int]int{}
	for rows.Next() {
		var gid, total int
		if err := rows.Scan(&gid, &total); err != nil {
			return err
		}
		totals[gid] = total
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range guests {
		total := totals[guests[i].ID]
		sisa := guests[i].Harga - total
		if sisa < 0 {
			sisa = 0
		}
		lunas := total >= guests[i].Harga
		guests[i].TotalBayar = &total
		guests[i].Sisa = &sisa
		guests[i].Lunas = &lunas
	}
	return nil
}

func adjustRoomOccupancy(tx *gorm.DB, roomID int, delta int) error {
	var room models.PenginapanRoom
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, roomID).Error; err != nil {
		return err
	}
	if room.Status == "maintenance" && delta > 0 {
		return errors.New("kamar dalam maintenance")
	}
	newOcc := room.CurrentOccupants + delta
	if newOcc < 0 {
		newOcc = 0
	}
	updates := map[string]interface{}{"current_occupants": newOcc}
	if room.Status != "maintenance" {
		if newOcc == 0 {
			updates["status"] = "kosong"
		} else {
			updates["status"] = "terisi"
		}
	}
	return tx.Model(&room).Updates(updates).Error
}

func ListGuests(c *gin.Context) {
	var data []models.PenginapanGuest
	q := db.PenginapanDB.Preload("Room").Order("id desc")
	switch c.Query("status") {
	case "inhouse":
		q = q.Where("checkout_date IS NULL")
	case "out":
		q = q.Where("checkout_date IS NOT NULL")
	}
	if v := c.Query("room_id"); v != "" {
		q = q.Where("room_id = ?", v)
	}
	if v := c.Query("checkin_date"); v != "" {
		q = q.Where("checkin_date = ?", v)
	}
	if err := q.Limit(500).Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil tamu", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	if err := attachGuestPaymentSummary(data); err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menghitung pembayaran", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Data tamu ditemukan", data, nil)
}

func GetGuest(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var row models.PenginapanGuest
	if err := db.PenginapanDB.Preload("Room").First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tamu tidak ditemukan", nil)
		return
	}
	guests := []models.PenginapanGuest{row}
	if err := attachGuestPaymentSummary(guests); err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menghitung pembayaran", nil)
		return
	}
	response.Success(c, http.StatusOK, "Tamu ditemukan", guests[0], nil)
}

func CheckinPenginapan(c *gin.Context) {
	var req request.PenginapanCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
		return
	}

	now := time.Now()
	checkinDate := now
	checkinTime := now

	guestRows := make([]models.PenginapanGuest, 0, len(req.Guests))
	roomDelta := map[int]int{}
	for i, g := range req.Guests {
		d, err := parsePenginapanDate(g.CheckinDate)
		if err != nil {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("guests[%d]: %v", i, err), nil)
			return
		}
		tm, err := parsePenginapanTime(g.CheckinTime)
		if err != nil {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("guests[%d]: %v", i, err), nil)
			return
		}
		cd := checkinDate
		ct := checkinTime
		if d != nil {
			cd = *d
		}
		if tm != nil {
			ct = *tm
		}
		ctStr := fmt.Sprintf("%02d:%02d:%02d", ct.Hour(), ct.Minute(), ct.Second())
		guestRows = append(guestRows, models.PenginapanGuest{
			RoomID:      g.RoomID,
			Name:        g.Name,
			Nik:         g.Nik,
			Gender:      g.Gender,
			Harga:       g.Harga,
			CheckinDate: &cd,
			CheckinTime: &ctStr,
			UserID:      penginapanUserID(c),
		})
		roomDelta[g.RoomID]++
	}

	allocations, err := buildCheckinAllocations(req.Payment, len(guestRows))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tx := db.PenginapanDB.Begin()
	if tx.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	defer tx.Rollback()

	for roomID, n := range roomDelta {
		var room models.PenginapanRoom
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, roomID).Error; err != nil {
			response.Error(c, http.StatusNotFound, fmt.Sprintf("room_id %d tidak ditemukan", roomID), nil)
			return
		}
		if room.Status == "maintenance" {
			response.Error(c, http.StatusConflict, fmt.Sprintf("kamar %s dalam maintenance", room.RoomNumber), nil)
			return
		}
		if room.CurrentOccupants+n > room.Capacity {
			response.Error(c, http.StatusConflict, fmt.Sprintf("kamar %s tidak mencukupi (sisa kapasitas %d)", room.RoomNumber, room.Capacity-room.CurrentOccupants), nil)
			return
		}
	}

	for i := range guestRows {
		if err := tx.Create(&guestRows[i]).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menyimpan tamu", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		history := models.RoomHistory{
			GuestID:    guestRows[i].ID,
			RoomID:     guestRows[i].RoomID,
			Status:     "check-in",
			OccurredAt: now,
			UserID:     penginapanUserID(c),
		}
		if err := tx.Create(&history).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menyimpan history check-in", nil)
			return
		}
	}

	var paymentOut *models.Payment
	if req.Payment != nil {
		payment, err := createPaymentTx(tx, req.Payment.Amount, req.Payment.PaymentMethodID, req.Payment.SourceID, req.Payment.Status, req.Payment.Note, penginapanUserID(c), now)
		if err != nil {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		for _, a := range allocations {
			pg := models.PaymentGuest{PaymentID: payment.ID, GuestID: guestRows[a.GuestIndex].ID, Amount: a.Amount}
			if err := tx.Create(&pg).Error; err != nil {
				response.Error(c, http.StatusInternalServerError, "Gagal menyimpan alokasi pembayaran", nil)
				return
			}
			payment.Guests = append(payment.Guests, pg)
		}
		paymentOut = payment
	}

	for roomID, n := range roomDelta {
		if err := adjustRoomOccupancy(tx, roomID, n); err != nil {
			response.Error(c, http.StatusConflict, err.Error(), nil)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal commit transaksi", nil)
		return
	}

	if err := attachGuestPaymentSummary(guestRows); err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menghitung pembayaran", nil)
		return
	}
	if paymentOut != nil {
		var full models.Payment
		if err := db.PenginapanDB.Preload("Source").Preload("PaymentMethod").First(&full, paymentOut.ID).Error; err == nil {
			paymentOut = &full
		}
	}

	out := gin.H{"guests": guestRows, "payment": paymentOut}
	guestIDs := make([]int, 0, len(guestRows))
	for _, g := range guestRows {
		guestIDs = append(guestIDs, g.ID)
	}
	for roomID := range roomDelta {
		broadcastRoomEvent("occupancy", roomID)
	}
	broadcastGuestEvent("checkin", guestIDs)
	logger.Info("CHECKIN PENGINAPAN jumlah_tamu=%d", len(guestRows))
	response.Success(c, http.StatusCreated, "Check-in berhasil", out, nil)
}

func buildCheckinAllocations(p *request.PenginapanPaymentInput, guestCount int) ([]request.PenginapanAllocation, error) {
	if p == nil {
		return nil, nil
	}
	if len(p.Allocations) == 0 {
		if guestCount == 1 {
			return []request.PenginapanAllocation{{GuestIndex: 0, Amount: p.Amount}}, nil
		}
		return nil, errors.New("payment.allocations wajib diisi untuk lebih dari 1 tamu")
	}
	total := 0
	for _, a := range p.Allocations {
		if a.GuestIndex < 0 || a.GuestIndex >= guestCount {
			return nil, fmt.Errorf("allocations: guest_index %d di luar rentang", a.GuestIndex)
		}
		if a.Amount < 0 {
			return nil, errors.New("allocations: amount tidak boleh negatif")
		}
		total += a.Amount
	}
	if total != p.Amount {
		return nil, fmt.Errorf("total allocations (%d) harus sama dengan amount pembayaran (%d)", total, p.Amount)
	}
	return p.Allocations, nil
}

func createPaymentTx(tx *gorm.DB, amount, methodID, sourceID int, status string, note *string, userID *uint, now time.Time) (*models.Payment, error) {
	var method models.PaymentMethod
	if err := tx.First(&method, methodID).Error; err != nil {
		return nil, errors.New("payment_method_id tidak ditemukan")
	}
	var source models.PurchaseSource
	if err := tx.First(&source, sourceID).Error; err != nil {
		return nil, errors.New("source_id tidak ditemukan")
	}
	if status == "" {
		status = "lunas"
	}
	if status != "lunas" && status != "pending" {
		return nil, errors.New("status pembayaran harus lunas atau pending")
	}
	payment := models.Payment{
		Amount:          amount,
		PaymentMethodID: methodID,
		SourceID:        sourceID,
		Status:          status,
		Note:            note,
		UserID:          userID,
	}
	if status == "lunas" {
		t := now
		payment.PaidAt = &t
	}
	if err := tx.Create(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func UpdateGuestPenginapan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Nik    *string `json:"nik"`
		Gender *string `json:"gender"`
		Harga  *int    `json:"harga"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}
	if req.Gender != nil && *req.Gender != "pria" && *req.Gender != "wanita" {
		response.Error(c, http.StatusBadRequest, "gender harus pria/wanita", nil)
		return
	}
	var row models.PenginapanGuest
	if err := db.PenginapanDB.First(&row, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tamu tidak ditemukan", nil)
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Nik != nil {
		updates["nik"] = *req.Nik
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Harga != nil {
		updates["harga"] = *req.Harga
	}
	if len(updates) == 0 {
		response.Error(c, http.StatusBadRequest, "tidak ada field untuk diupdate", nil)
		return
	}
	if err := db.PenginapanDB.Model(&row).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal update tamu", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "Tamu berhasil diupdate", row, nil)
}

func CheckoutPenginapan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req request.PenginapanCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}

	now := time.Now()
	checkoutDate := now
	checkoutTime := now
	if d, err := parsePenginapanDate(req.CheckoutDate); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	} else if d != nil {
		checkoutDate = *d
	}
	if t, err := parsePenginapanTime(req.CheckoutTime); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	} else if t != nil {
		checkoutTime = *t
	}
	coStr := fmt.Sprintf("%02d:%02d:%02d", checkoutTime.Hour(), checkoutTime.Minute(), checkoutTime.Second())

	tx := db.PenginapanDB.Begin()
	if tx.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	defer tx.Rollback()

	var guest models.PenginapanGuest
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&guest, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tamu tidak ditemukan", nil)
		return
	}
	if guest.CheckoutDate != nil {
		response.Error(c, http.StatusConflict, "Tamu sudah check-out", nil)
		return
	}

	if err := tx.Model(&guest).Updates(map[string]interface{}{
		"checkout_date": checkoutDate,
		"checkout_time": coStr,
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal update tamu", nil)
		return
	}

	history := models.RoomHistory{
		GuestID:    guest.ID,
		RoomID:     guest.RoomID,
		Status:     "check-out",
		OccurredAt: now,
		UserID:     penginapanUserID(c),
	}
	if err := tx.Create(&history).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menyimpan history check-out", nil)
		return
	}

	if err := adjustRoomOccupancy(tx, guest.RoomID, -1); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if err := tx.Commit().Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal commit transaksi", nil)
		return
	}

	logger.Info("CHECKOUT PENGINAPAN guest_id=%d room_id=%d", guest.ID, guest.RoomID)
	broadcastRoomEvent("occupancy", guest.RoomID)
	broadcastGuestEvent("checkout", []int{guest.ID})
	response.Success(c, http.StatusOK, "Check-out berhasil", nil, nil)
}

func PindahKamarPenginapan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var req request.PenginapanPindahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
		return
	}

	now := time.Now()
	tx := db.PenginapanDB.Begin()
	if tx.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	defer tx.Rollback()

	var guest models.PenginapanGuest
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&guest, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tamu tidak ditemukan", nil)
		return
	}
	if guest.CheckoutDate != nil {
		response.Error(c, http.StatusConflict, "Tamu sudah check-out", nil)
		return
	}
	if guest.RoomID == req.RoomID {
		response.Error(c, http.StatusBadRequest, "room_id tujuan sama dengan kamar sekarang", nil)
		return
	}

	var oldRoom, newRoom models.PenginapanRoom
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&oldRoom, guest.RoomID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Kamar asal tidak ditemukan", nil)
		return
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&newRoom, req.RoomID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Kamar tujuan tidak ditemukan", nil)
		return
	}
	if newRoom.Status == "maintenance" {
		response.Error(c, http.StatusConflict, "kamar tujuan dalam maintenance", nil)
		return
	}
	if newRoom.CurrentOccupants+1 > newRoom.Capacity {
		response.Error(c, http.StatusConflict, fmt.Sprintf("kamar tujuan %s tidak mencukupi", newRoom.RoomNumber), nil)
		return
	}

	if err := tx.Model(&guest).Update("room_id", req.RoomID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal update kamar tamu", nil)
		return
	}

	note := fmt.Sprintf("%s -> %s", oldRoom.RoomNumber, newRoom.RoomNumber)
	history := models.RoomHistory{
		GuestID:    guest.ID,
		RoomID:     req.RoomID,
		Status:     "pindah",
		OccurredAt: now,
		UserID:     penginapanUserID(c),
		Note:       &note,
	}
	if err := tx.Create(&history).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menyimpan history pindah", nil)
		return
	}

	if err := adjustRoomOccupancy(tx, oldRoom.ID, -1); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if err := adjustRoomOccupancy(tx, newRoom.ID, 1); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if err := tx.Commit().Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal commit transaksi", nil)
		return
	}

	logger.Info("PINDAH KAMAR PENGINAPAN guest_id=%d %s", guest.ID, note)
	broadcastRoomEvent("occupancy", oldRoom.ID)
	broadcastRoomEvent("occupancy", newRoom.ID)
	broadcastGuestEvent("pindah", []int{guest.ID})
	response.Success(c, http.StatusOK, "Pindah kamar berhasil", gin.H{"guest_id": guest.ID, "room_lama": oldRoom.ID, "room_baru": newRoom.ID}, nil)
}

func ListGuestHistories(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var guest models.PenginapanGuest
	if err := db.PenginapanDB.First(&guest, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tamu tidak ditemukan", nil)
		return
	}
	var data []models.RoomHistory
	if err := db.PenginapanDB.Where("guest_id = ?", id).Order("id asc").Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil history tamu", nil)
		return
	}
	response.Success(c, http.StatusOK, "History tamu ditemukan", data, nil)
}

func ListRoomHistories(c *gin.Context) {
	var data []models.RoomHistory
	q := db.PenginapanDB.Order("id desc").Limit(500)
	if v := c.Query("room_id"); v != "" {
		q = q.Where("room_id = ?", v)
	}
	if v := c.Query("guest_id"); v != "" {
		q = q.Where("guest_id = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if err := q.Find(&data).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil history kamar", nil)
		return
	}
	response.Success(c, http.StatusOK, "History kamar ditemukan", data, nil)
}
