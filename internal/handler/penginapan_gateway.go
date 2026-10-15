package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/payment"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func gatewayConfigValue(key string) (string, bool) {
	var cfg models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", key).First(&cfg).Error; err != nil {
		return "", false
	}
	if cfg.ValueConfigs == nil {
		return "", false
	}
	v := strings.TrimSpace(*cfg.ValueConfigs)
	return v, v != ""
}

func gatewaySettingsFor(provider string) payment.Settings {
	s := payment.Settings{CallbackURL: func() string { v, _ := gatewayConfigValue("payment_gateway_callback_url"); return v }()}
	switch provider {
	case "midtrans":
		s.ServerKey, _ = gatewayConfigValue("midtrans_server_key")
		s.Env, _ = gatewayConfigValue("midtrans_env")
		s.BaseURL, _ = gatewayConfigValue("midtrans_api_url")
	case "xendit":
		s.ServerKey, _ = gatewayConfigValue("xendit_secret_key")
		s.CallbackToken, _ = gatewayConfigValue("xendit_callback_token")
		s.Env, _ = gatewayConfigValue("xendit_env")
		s.BaseURL, _ = gatewayConfigValue("xendit_api_url")
	}
	return s
}

func loadGatewayByName(provider string) (payment.Gateway, error) {
	if provider == "" {
		return nil, payment.ErrNotConfigured
	}
	return payment.New(provider, gatewaySettingsFor(provider))
}

func loadActiveGateway() (payment.Gateway, string, error) {
	provider, ok := gatewayConfigValue("payment_gateway")
	if !ok {
		return nil, "", payment.ErrNotConfigured
	}
	provider = strings.ToLower(provider)
	gw, err := loadGatewayByName(provider)
	if err != nil {
		return nil, provider, err
	}
	return gw, provider, nil
}

func GetPaymentGateway(c *gin.Context) {
	provider, ok := gatewayConfigValue("payment_gateway")
	data := gin.H{"provider": "", "configured": false}
	if ok {
		provider = strings.ToLower(provider)
		data["provider"] = provider
		data["env"], _ = gatewayConfigValue(provider + "_env")
		data["webhook_path"] = "/penginapan/webhooks/" + provider
		if _, err := loadGatewayByName(provider); err == nil {
			data["configured"] = true
		} else {
			data["detail"] = err.Error()
		}
	} else {
		data["detail"] = "isi key_configs 'payment_gateway' (midtrans|xendit) + kredensialnya via POST /penginapan/configs"
	}
	response.Success(c, http.StatusOK, "Konfigurasi payment gateway", data, nil)
}

func PayWithPayment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}
	var pay models.Payment
	if err := db.PenginapanDB.Preload("Guests.Guest").First(&pay, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}
	if pay.Status == "batal" {
		response.Error(c, http.StatusConflict, "Pembayaran sudah dibatalkan", []response.ErrorItem{{Code: "PAYMENT_VOID", Detail: "buat pembayaran baru untuk tagihan gateway"}})
		return
	}
	if pay.Status == "lunas" {
		response.Error(c, http.StatusConflict, "Pembayaran sudah lunas", []response.ErrorItem{{Code: "PAYMENT_PAID", Detail: "tidak perlu tagihan gateway"}})
		return
	}
	gw, provider, err := loadActiveGateway()
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment gateway belum dikonfigurasi", []response.ErrorItem{{
			Code:   "NOT_CONFIGURED",
			Detail: "set key_configs 'payment_gateway' + kredensial via POST /penginapan/configs (" + err.Error() + ")",
		}})
		return
	}
	if pay.ExternalID != nil && pay.QRString != nil && *pay.QRString != "" &&
		pay.Gateway == provider && pay.GatewayStatus != nil && *pay.GatewayStatus == "pending" &&
		(pay.GatewayExpiresAt == nil || pay.GatewayExpiresAt.After(time.Now())) {
		response.Success(c, http.StatusOK, "QRIS pembayaran sudah ada", gin.H{
			"payment_id":     pay.ID,
			"provider":       pay.Gateway,
			"external_id":    *pay.ExternalID,
			"qr_string":      *pay.QRString,
			"gateway_status": *pay.GatewayStatus,
			"expires_at":     pay.GatewayExpiresAt,
		}, nil)
		return
	}

	extID := "TPG-PAY-" + strconv.Itoa(pay.ID) + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	name := ""
	if len(pay.Guests) > 0 && pay.Guests[0].Guest != nil {
		name = pay.Guests[0].Guest.Name
	}
	res, err := gw.Charge(payment.ChargeRequest{
		ExternalID:   extID,
		Amount:       pay.Amount,
		Description:  "Pembayaran penginapan #" + strconv.Itoa(pay.ID),
		CustomerName: name,
		Type:         "qris",
	})
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Payment gateway gagal membuat QRIS", []response.ErrorItem{{Code: "GATEWAY_ERROR", Detail: err.Error()}})
		return
	}

	gatewayStatus := "pending"
	updates := map[string]any{
		"gateway":            provider,
		"external_id":        res.ExternalID,
		"qr_string":          res.QRString,
		"gateway_status":     gatewayStatus,
		"gateway_expires_at": res.ExpiresAt,
	}
	if err := db.PenginapanDB.Model(&models.Payment{}).Where("id = ?", pay.ID).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal menyimpan QRIS gateway", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
		return
	}
	response.Success(c, http.StatusOK, "QRIS pembayaran dibuat", gin.H{
		"payment_id":     pay.ID,
		"provider":       provider,
		"external_id":    res.ExternalID,
		"qr_string":      res.QRString,
		"gateway_status": gatewayStatus,
		"expires_at":     res.ExpiresAt,
	}, nil)
}

func WebhookMidtrans(c *gin.Context) { gatewayWebhook(c, "midtrans") }
func WebhookXendit(c *gin.Context)   { gatewayWebhook(c, "xendit") }

func gatewayWebhook(c *gin.Context, provider string) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Gagal membaca body webhook", []response.ErrorItem{{Code: "BAD_REQUEST", Detail: err.Error()}})
		return
	}
	gw, err := loadGatewayByName(provider)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Payment gateway belum dikonfigurasi", []response.ErrorItem{{Code: "NOT_CONFIGURED", Detail: err.Error()}})
		return
	}
	res, err := gw.VerifyWebhook(c.Request, body)
	if err != nil {
		if errors.Is(err, payment.ErrInvalidSignature) {
			response.Error(c, http.StatusUnauthorized, "Signature/token webhook tidak valid", []response.ErrorItem{{Code: "INVALID_SIGNATURE", Detail: err.Error()}})
			return
		}
		response.Error(c, http.StatusBadRequest, "Webhook tidak valid", []response.ErrorItem{{Code: "BAD_REQUEST", Detail: err.Error()}})
		return
	}

	var pay models.Payment
	if err := db.PenginapanDB.Where("external_id = ?", res.ExternalID).First(&pay).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran dengan external_id tersebut tidak ditemukan", []response.ErrorItem{{Code: "PAYMENT_NOT_FOUND", Detail: res.ExternalID}})
		return
	}
	if res.Amount > 0 && res.Amount != pay.Amount {
		response.Error(c, http.StatusForbidden, "Jumlah pembayaran tidak sesuai", []response.ErrorItem{{Code: "AMOUNT_MISMATCH", Detail: "gateway=" + strconv.Itoa(res.Amount) + " pembayaran=" + strconv.Itoa(pay.Amount)}})
		return
	}

	switch res.Status {
	case "paid":
		if pay.Status == "lunas" {
			response.Success(c, http.StatusOK, "Webhook diproses", gin.H{"status": "ok", "payment_id": pay.ID, "payment_status": "lunas", "already_paid": true}, nil)
			return
		}
		paidAt := time.Now()
		if res.PaidAt != nil {
			paidAt = *res.PaidAt
		}
		updates := map[string]any{"status": "lunas", "paid_at": paidAt, "gateway_status": res.RawStatus}
		if err := db.PenginapanDB.Model(&models.Payment{}).Where("id = ?", pay.ID).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menandai pembayaran lunas", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		response.Success(c, http.StatusOK, "Webhook diproses", gin.H{"status": "ok", "payment_id": pay.ID, "payment_status": "lunas"}, nil)
	case "failed", "expired":
		db.PenginapanDB.Model(&models.Payment{}).Where("id = ?", pay.ID).Updates(map[string]any{"gateway_status": res.RawStatus})
		response.Success(c, http.StatusOK, "Webhook diproses", gin.H{"status": "ok", "payment_id": pay.ID, "payment_status": pay.Status, "gateway_status": res.RawStatus}, nil)
	default:
		db.PenginapanDB.Model(&models.Payment{}).Where("id = ?", pay.ID).Updates(map[string]any{"gateway_status": res.RawStatus})
		response.Success(c, http.StatusOK, "Webhook diproses", gin.H{"status": "ok", "payment_id": pay.ID, "payment_status": pay.Status, "gateway_status": res.RawStatus}, nil)
	}
}
