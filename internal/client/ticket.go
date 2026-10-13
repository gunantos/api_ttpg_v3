package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TicketAPIResponse struct {
	ST  bool       `json:"st"`
	Obj TicketData `json:"obj"`
}

type TicketData struct {
	NoKursi          string `json:"no_kursi"`
	TanggalBerangkat string `json:"tanggal_berangkat"`
	JamBerangkat     string `json:"jam_berangkat"`
	NoBarcode        string `json:"no_barcode"`
	NamaPenumpang    string `json:"nama_penumpang"`
	NoKontak         string `json:"no_kontak"`
	NoKTP            string `json:"no_ktp"`
	Turun            string `json:"turun"`
	NoBody           string `json:"no_body"`
	NoPolisi         string `json:"no_polisi"`
}

type TicketClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTicketClient() *TicketClient {
	return &TicketClient{
		BaseURL: "https://newticket.dgti.co.id",
		HTTPClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (tc *TicketClient) GetBarcodeData(noreg string) (*TicketData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return tc.GetBarcodeDataWithContext(ctx, noreg)
}

func (tc *TicketClient) GetBarcodeDataWithContext(ctx context.Context, noreg string) (*TicketData, error) {
	url := fmt.Sprintf("%s/index.php/api/barcode/%s", tc.BaseURL, noreg)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp TicketAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.ST {
		return nil, fmt.Errorf("API returned false status for noreg: %s", noreg)
	}

	return &apiResp.Obj, nil
}
