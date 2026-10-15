package payment

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotConfigured    = errors.New("payment gateway belum dikonfigurasi")
	ErrUnknownProvider  = errors.New("payment gateway tidak dikenal")
	ErrInvalidSignature = errors.New("signature/token webhook tidak valid")
)

type Settings struct {
	BaseURL       string
	ServerKey     string
	CallbackToken string
	Env           string
	CallbackURL   string
}

type ChargeRequest struct {
	ExternalID    string
	Amount        int
	Description   string
	CustomerName  string
	CustomerEmail string
	Type          string // qris (default) | va (belum didukung provider saat ini)
	Bank          string // untuk Type=va nanti (bca/mandiri/bni/…)
}

type ChargeResult struct {
	Provider   string
	ExternalID string
	QRString   string
	InvoiceURL string
	SnapToken  string
	ExpiresAt  *time.Time
}

type WebhookResult struct {
	ExternalID string
	Status     string // paid | pending | failed | expired
	RawStatus  string
	Amount     int
	PaidAt     *time.Time
}

type Gateway interface {
	Name() string
	Charge(req ChargeRequest) (*ChargeResult, error)
	VerifyWebhook(r *http.Request, body []byte) (*WebhookResult, error)
}

type Factory func(Settings) (Gateway, error)

var registry = map[string]Factory{}

// Register dipanggil init() tiap gateway — provider baru cukup tambah file + Register.
func Register(name string, f Factory) {
	registry[strings.ToLower(strings.TrimSpace(name))] = f
}

func New(name string, s Settings) (Gateway, error) {
	f := registry[strings.ToLower(strings.TrimSpace(name))]
	if f == nil {
		return nil, ErrUnknownProvider
	}
	return f(s)
}

func Providers() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
