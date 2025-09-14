package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Bypass  string // optional vercel bypass token
	h       *http.Client
	Email   string
	DeviceID string
}

func New(baseURL, bypass, email, deviceID string) *Client {
	return &Client{
		BaseURL: baseURL,
		Bypass:  bypass,
		Email:   email,
		DeviceID: deviceID,
		h: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) url(p string) string {
	u := c.BaseURL + p
	if c.Bypass != "" {
		sep := "?"
		if bytes.Contains([]byte(u), []byte("?")) { sep = "&" }
		u = fmt.Sprintf("%s%sx-vercel-set-bypass-cookie=true&x-vercel-protection-bypass=%s", u, sep, c.Bypass)
	}
	return u
}

func (c *Client) Register(ctx context.Context, referral, version string) error {
	body := map[string]any{
		"email":    c.Email,
		"deviceId": c.DeviceID,
		"referral": referral,
		"version":  version,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.url("/api/agent/register"), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.h.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("register: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Beat(ctx context.Context) error {
	body := map[string]any{ "email": c.Email, "deviceId": c.DeviceID }
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.url("/api/agent/beat"), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.h.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("beat: status %d", resp.StatusCode)
	}
	return nil
}

type NextJobResp struct {
	JobID      string          `json:"jobId"`
	Type       string          `json:"type"`
	Args       json.RawMessage `json:"args"`
	TimeoutSec int             `json:"timeoutSec"`
}

func (c *Client) NextJob(ctx context.Context) (*NextJobResp, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		c.url(fmt.Sprintf("/api/agent/jobs/next?email=%s&deviceId=%s", c.Email, c.DeviceID)), nil)
	resp, err := c.h.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode == 204 { return nil, nil }
	if resp.StatusCode/100 != 2 { return nil, fmt.Errorf("next job: %d", resp.StatusCode) }
	var out NextJobResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
	return &out, nil
}

func (c *Client) Report(ctx context.Context, jobID, status string, dur time.Duration, errMsg string) error {
	body := map[string]any{
		"jobId": jobID, "status": status, "durationMs": dur.Milliseconds(), "error": errMsg,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.url("/api/agent/jobs/report"), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.h.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 { return fmt.Errorf("report: %d", resp.StatusCode) }
	return nil
}
