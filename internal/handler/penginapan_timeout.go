package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"gorm.io/gorm/clause"
)

const paymentHoldDefaultMinutes = 15

// paymentHoldMinutes membaca config DB `payment_hold_minutes` (menit).
// Absen/tidak valid → default 15; 0 = fitur nonaktif.
func paymentHoldMinutes() int {
	var cfg models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", "payment_hold_minutes").First(&cfg).Error; err != nil ||
		cfg.ValueConfigs == nil {
		return paymentHoldDefaultMinutes
	}
	v, err := strconv.Atoi(strings.TrimSpace(*cfg.ValueConfigs))
	if err != nil {
		return paymentHoldDefaultMinutes
	}
	if v < 0 {
		return 0
	}
	return v
}

// sweepPendingCashPayments membatalkan payment cash/manual berstatus pending
// yang lebih tua dari hold: payment → 'batal' + tamu terkait yang belum lunas
// di-check-out otomatis (kamar kembali available). Payment yang sudah di-charge
// gateway (kolom gateway terisi) TIDAK disentuh — expiry-nya urusan provider.
func sweepPendingCashPayments(now time.Time) {
	hold := paymentHoldMinutes()
	if hold <= 0 || db.PenginapanDB == nil {
		return
	}
	cutoff := now.Add(-time.Duration(hold) * time.Minute)
	var pays []models.Payment
	if err := db.PenginapanDB.Model(&models.Payment{}).
		Joins("JOIN payment_methods pm ON pm.id = payments.payment_method_id").
		Where("payments.status = ?", "pending").
		Where("payments.created_at < ?", cutoff).
		Where("pm.code = ?", "cash").
		Where("(payments.gateway IS NULL OR payments.gateway = '')").
		Find(&pays).Error; err != nil {
		logger.Error("payment timeout sweeper: gagal baca pending cash: %v", err)
		return
	}
	for i := range pays {
		voidTimedOutPayment(&pays[i], now)
	}
}

func voidTimedOutPayment(pay *models.Payment, now time.Time) {
	tx := db.PenginapanDB.Begin()
	if tx.Error != nil {
		logger.Error("payment timeout: gagal mulai transaksi payment_id=%d: %v", pay.ID, tx.Error)
		return
	}
	defer tx.Rollback()

	res := tx.Model(&models.Payment{}).
		Where("id = ? AND status = 'pending'", pay.ID).
		Update("status", "batal")
	if res.Error != nil {
		logger.Error("payment timeout: gagal void payment_id=%d: %v", pay.ID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		return
	}

	var guestIDs []int
	if err := tx.Model(&models.PaymentGuest{}).Where("payment_id = ?", pay.ID).Pluck("guest_id", &guestIDs).Error; err != nil {
		logger.Error("payment timeout: gagal baca alokasi payment_id=%d: %v", pay.ID, err)
		return
	}

	coStr := fmt.Sprintf("%02d:%02d:%02d", now.Hour(), now.Minute(), now.Second())
	note := "check-out otomatis: pembayaran timeout"
	var outGuests []int
	var outRooms []int

	for _, gid := range guestIDs {
		var g models.PenginapanGuest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, gid).Error; err != nil {
			continue
		}
		if g.CheckoutDate != nil {
			continue
		}
		var lunasSum int
		if err := tx.Raw("SELECT COALESCE(SUM(pg.amount),0) FROM payment_guests pg JOIN payments p ON p.id = pg.payment_id WHERE pg.guest_id = ? AND p.status = 'lunas'", gid).Scan(&lunasSum).Error; err != nil {
			logger.Error("payment timeout: gagal cek lunas guest_id=%d: %v", gid, err)
			continue
		}
		if lunasSum >= g.Harga {
			continue
		}
		if err := tx.Model(&g).Updates(map[string]interface{}{
			"checkout_date": now,
			"checkout_time": coStr,
		}).Error; err != nil {
			logger.Error("payment timeout: gagal checkout guest_id=%d: %v", gid, err)
			return
		}
		history := models.RoomHistory{
			GuestID:    g.ID,
			RoomID:     g.RoomID,
			Status:     "check-out",
			OccurredAt: now,
			Note:       &note,
		}
		if err := tx.Create(&history).Error; err != nil {
			logger.Error("payment timeout: gagal history guest_id=%d: %v", gid, err)
			return
		}
		if err := adjustRoomOccupancy(tx, g.RoomID, -1); err != nil {
			logger.Error("payment timeout: gagal occupancy room_id=%d: %v", g.RoomID, err)
			return
		}
		outGuests = append(outGuests, g.ID)
		outRooms = append(outRooms, g.RoomID)
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("payment timeout: gagal commit payment_id=%d: %v", pay.ID, err)
		return
	}

	logger.Warning("PAYMENT TIMEOUT: payment_id=%d amount=%d → batal; check-out guest=%v room=%v", pay.ID, pay.Amount, outGuests, outRooms)
	for _, rid := range outRooms {
		broadcastRoomEvent("occupancy", rid)
	}
	if len(outGuests) > 0 {
		broadcastGuestEvent("checkout", outGuests)
	}
}

// StartPaymentTimeoutSweeper jalan di goroutine terpisah (dipanggil main.go):
// sweep pertama 10 detik setelah start, lalu tiap 60 detik.
func StartPaymentTimeoutSweeper() {
	time.Sleep(10 * time.Second)
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("payment timeout sweeper panic: %v", r)
				}
			}()
			sweepPendingCashPayments(time.Now())
		}()
		time.Sleep(time.Minute)
	}
}
