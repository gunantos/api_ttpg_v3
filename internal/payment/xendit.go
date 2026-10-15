package payment

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Xendit struct{ settings Settings }

func NewXendit(s Settings) (Gateway, error) {
	if s.BaseURL == "" {
		s.BaseURL = "https://api.xendit.co"
	}
	if s.ServerKey == "" {
		return nil, ErrNotConfigured
	}
	return &Xendit{settings: s}, nil
}

func init() { Register("xendit", NewXendit) }

func (g *Xendit) Name() string { return "xendit" }

func (g *Xendit) Charge(req ChargeRequest) (*ChargeResult, error) {
	if req.Type == "va" {
		return nil, fmt.Errorf("xendit: VA belum didukung, fokus QRIS")
	}
	payload := map[string]any{
		"external_id":    req.ExternalID,
		"amount":         req.Amount,
		"type":           "DYNAMIC",
		"payment_method": "QRIS",
		"description":    req.Description,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(g.settings.BaseURL, "/")+"/qr-codes", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-SECRET-KEY", g.settings.ServerKey)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xendit tidak terjangkau: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := e.Message
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return nil, fmt.Errorf("xendit HTTP %d: %s", resp.StatusCode, msg)
	}

	var out struct {
		QRString          string `json:"qr_string"`
		ExpiresAt         string `json:"expires_at"`
		ChannelProperties struct {
			QRString string `json:"qr_string"`
		} `json:"channel_properties"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("respons xendit tidak valid: %w", err)
	}
	qr := out.QRString
	if qr == "" {
		qr = out.ChannelProperties.QRString
	}
	if qr == "" {
		return nil, fmt.Errorf("xendit tidak mengembalikan qr_string")
	}
	res := &ChargeResult{
		Provider:   "xendit",
		ExternalID: req.ExternalID,
		QRString:   qr,
	}
	if out.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, out.ExpiresAt); err == nil {
			res.ExpiresAt = &t
		}
	}
	return res, nil
}

func (g *Xendit) VerifyWebhook(r *http.Request, body []byte) (*WebhookResult, error) {
	token := r.Header.Get("X-Callback-Token")
	if g.settings.CallbackToken == "" || token == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(g.settings.CallbackToken)) != 1 {
		return nil, ErrInvalidSignature
	}

	var n struct {
		ExternalID  string          `json:"external_id"`
		ReferenceID string          `json:"reference_id"`
		Status      string          `json:"status"`
		PaidAt      string          `json:"paid_at"`
		Amount      json.RawMessage `json:"amount"`
	}
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("body xendit tidak valid: %w", err)
	}
	if n.ExternalID == "" {
		n.ExternalID = n.ReferenceID
	}
	if n.ExternalID == "" {
		return nil, ErrInvalidSignature
	}

	res := &WebhookResult{
		ExternalID: n.ExternalID,
		RawStatus:  n.Status,
		Status:     "pending",
	}
	if v, err := strconv.Atoi(rawJSONString(n.Amount)); err == nil {
		res.Amount = v
	}
	switch strings.ToUpper(n.Status) {
	case "PAID", "SETTLED":
		res.Status = "paid"
		now := time.Now()
		res.PaidAt = &now
		if n.PaidAt != "" {
			if t, err := time.Parse(time.RFC3339, n.PaidAt); err == nil {
				res.PaidAt = &t
			}
		}
	case "FAILED":
		res.Status = "failed"
	case "EXPIRED":
		res.Status = "expired"
	}
	return res, nil
}
