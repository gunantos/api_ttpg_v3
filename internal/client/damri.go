package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrDamriTicketNotFound = errors.New("tiket damri tidak ditemukan")

type DamriClient struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client

	mu          sync.Mutex
	token       string
	tokenExpire time.Time
}

type DamriScanData struct {
	TicketID        string `json:"ticket_id"`
	DateOfDeparture string `json:"date_of_departure"`
	TimeOfDeparture string `json:"time_of_departure"`
	DestinationCode string `json:"destination_code"`
	DestinationName string `json:"destination_name"`
	PassengerName   string `json:"passenger_name"`
	Nopol           string `json:"nopol"`
}

type damriTokenResponse struct {
	Token string `json:"token"`
}

type damriScanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    []struct {
		Nama     string `json:"nama"`
		Tanggal  string `json:"tanggal"`
		Jam      string `json:"jam"`
		Tujuan   string `json:"tujuan"`
		NMTujuan string `json:"nm_tujuan"`
		Nopol    string `json:"no_polisi"`
	} `json:"data"`
}

func NewDamriClient(baseURL, username, password string, timeout time.Duration) *DamriClient {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &DamriClient{
		BaseURL:  strings.TrimRight(baseURL, "/") + "/",
		Username: username,
		Password: password,
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

func (c *DamriClient) ScanTicket(ctx context.Context, ticketID string) (*DamriScanData, error) {
	token, err := c.getToken(ctx, false)
	if err != nil {
		return nil, err
	}
	data, err := c.scanTicket(ctx, token, ticketID)
	if err == nil {
		return data, nil
	}
	c.mu.Lock()
	c.token = ""
	c.tokenExpire = time.Time{}
	c.mu.Unlock()
	token, tokenErr := c.getToken(ctx, true)
	if tokenErr != nil {
		return nil, err
	}
	return c.scanTicket(ctx, token, ticketID)
}

func (c *DamriClient) getToken(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	if !force && c.token != "" && time.Now().Before(c.tokenExpire) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()
	form := url.Values{}
	form.Set("username", c.Username)
	form.Set("password", c.Password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"authtimeout/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("damri token status %d", resp.StatusCode)
	}
	var tokenResp damriTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Token == "" {
		return "", errors.New("damri token kosong")
	}
	c.mu.Lock()
	c.token = tokenResp.Token
	c.tokenExpire = time.Now().Add(20 * time.Minute)
	c.mu.Unlock()
	return tokenResp.Token, nil
}

func (c *DamriClient) scanTicket(ctx context.Context, token, ticketID string) (*DamriScanData, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("kd_tiket", ticketID); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"v2_1/api/scantiket", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("damri scan status %d", resp.StatusCode)
	}
	var scanResp damriScanResponse
	if err := json.Unmarshal(respBody, &scanResp); err != nil {
		return nil, err
	}
	if strings.ToLower(scanResp.Status) != "success" || len(scanResp.Data) == 0 {
		if scanResp.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrDamriTicketNotFound, scanResp.Message)
		}
		return nil, ErrDamriTicketNotFound
	}
	first := scanResp.Data[0]
	return &DamriScanData{
		TicketID:        ticketID,
		DateOfDeparture: first.Tanggal,
		TimeOfDeparture: first.Jam,
		DestinationCode: first.Tujuan,
		DestinationName: first.NMTujuan,
		PassengerName:   first.Nama,
		Nopol:           first.Nopol,
	}, nil
}
