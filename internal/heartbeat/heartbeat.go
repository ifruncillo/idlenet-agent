package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Version string
	HTTP    *http.Client
}

func NewClient(baseURL, version string) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8787" // sensible default for local stub
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Version: version,
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type registerReq struct {
	Email    string `json:"email"`
	DeviceID string `json:"deviceId"`
	Referral string `json:"referral,omitempty"`
	Version  string `json:"version,omitempty"`
}

type beatReq struct {
	Email    string `json:"email"`
	DeviceID string `json:"deviceId"`
}

func (c *Client) Register(ctx context.Context, email, deviceID, referral string) error {
	body := registerReq{
		Email:    strings.TrimSpace(email),
		DeviceID: deviceID,
		Referral: strings.TrimSpace(referral),
		Version:  c.Version,
	}
	return c.post(ctx, "/api/agent/register", body)
}

func (c *Client) Beat(ctx context.Context, email, deviceID string) error {
	body := beatReq{
		Email:    strings.TrimSpace(email),
		DeviceID: deviceID,
	}
	return c.post(ctx, "/api/agent/beat", body)
}

func (c *Client) post(ctx context.Context, path string, payload any) error {
	u := c.BaseURL + path
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("IdleNet-Agent/%s (%s/%s)", orDefault(c.Version, "dev"), runtime.GOOS, runtime.GOARCH))

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("POST %s -> %s: %s", path, resp.Status, strings.TrimSpace(string(slurp)))
	}
	return nil
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
