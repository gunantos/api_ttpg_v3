package payment

import (
	"bytes"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Midtrans struct{ settings Settings }

func NewMidtrans(s Settings) (Gateway, error) {
	if strings.EqualFold(s.Env, "production") {
		if s.BaseURL == "" {
			s.BaseURL = "https://api.midtrans.com"
		}
	} else if s.BaseURL == "" {
		s.BaseURL = "https://api.sandbox.midtrans.com"
	}
	if s.ServerKey == "" {
		return nil, ErrNotConfigured
	}
	return &Midtrans{settings: s}, nil
}

func init() { Register("midtrans", NewMidtrans) }

func (g *Midtrans) Name() string { return "midtrans" }

func (g *Midtrans) Charge(req ChargeRequest) (*ChargeResult, error) {
	if req.Type == "va" {
		return nil, fmt.Errorf("midtrans: VA belum didukung, fokus QRIS")
	}
	payload := map[string]any{
		"payment_type": "qris",
		"transaction_details": map[string]any{
			"order_id":     req.ExternalID,
			"gross_amount": req.Amount,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(g.settings.BaseURL, "/")+"/v2/charge", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(g.settings.ServerKey, "")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("midtrans tidak terjangkau: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		var e struct {
			ErrorMessages []string `json:"error_messages"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := strings.Join(e.ErrorMessages, "; ")
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return nil, fmt.Errorf("midtrans HTTP %d: %s", resp.StatusCode, msg)
	}

	var out struct {
		QRString   string `json:"qr_string"`
		ExpiryTime string `json:"expiry_time"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("respons midtrans tidak valid: %w", err)
	}
	if out.QRString == "" {
		return nil, fmt.Errorf("midtrans tidak mengembalikan qr_string")
	}
	res := &ChargeResult{
		Provider:   "midtrans",
		ExternalID: req.ExternalID,
		QRString:   out.QRString,
	}
	if out.ExpiryTime != "" {
		for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
			if t, err := time.Parse(layout, out.ExpiryTime); err == nil {
				res.ExpiresAt = &t
				break
			}
		}
	}
	return res, nil
}

func rawJSONString(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, `"`)
	return strings.TrimSpace(s)
}

func (g *Midtrans) VerifyWebhook(r *http.Request, body []byte) (*WebhookResult, error) {
	var n struct {
		OrderID           string          `json:"order_id"`
		GrossAmount       json.RawMessage `json:"gross_amount"`
		StatusCode        string          `json:"status_code"`
		SignatureKey      string          `json:"signature_key"`
		TransactionStatus string          `json:"transaction_status"`
		SettlementTime    string          `json:"settlement_time"`
		CaptureTime       string          `json:"capture_time"`
	}
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("body midtrans tidak valid: %w", err)
	}
	if n.OrderID == "" || n.SignatureKey == "" {
		return nil, ErrInvalidSignature
	}
	gross := rawJSONString(n.GrossAmount)
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + gross + g.settings.ServerKey))
	expected := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(n.SignatureKey)), []byte(expected)) != 1 {
		return nil, ErrInvalidSignature
	}

	res := &WebhookResult{
		ExternalID: n.OrderID,
		RawStatus:  n.TransactionStatus,
		Status:     "pending",
	}
	if v, err := strconv.Atoi(strings.SplitN(gross, ".", 2)[0]); err == nil {
		res.Amount = v
	}
	switch n.TransactionStatus {
	case "settlement", "capture":
		res.Status = "paid"
		now := time.Now()
		for _, ts := range []string{n.SettlementTime, n.CaptureTime} {
			if ts != "" {
				if t, err := time.Parse("2006-01-02 15:04:05", ts); err == nil {
					res.PaidAt = &t
					break
				}
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					res.PaidAt = &t
					break
				}
			}
		}
		if res.PaidAt == nil {
			res.PaidAt = &now
		}
	case "cancel", "deny":
		res.Status = "failed"
	case "expire":
		res.Status = "expired"
	}
	return res, nil
}
