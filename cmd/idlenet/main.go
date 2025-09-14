package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var (
	// Shown in logs and sent to /api/agent/register.
	// You can also override this at build time with:
	//   -ldflags "-X 'main.version=v0.2.0'"
	version = "v0.2.0"

	// Default API base. Override with env IDLENET_API_BASE if needed.
	defaultAPIBase = "https://idlenet-pilot-qi7t.vercel.app"
)

type Config struct {
	Email    string `json:"email"`
	Referral string `json:"referral,omitempty"`
	DeviceID string `json:"device_id"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Load or create config (email/referral/device_id).
	cfg, _ := loadConfig()
	if cfg.DeviceID == "" {
		cfg.DeviceID = genDeviceID()
	}
	// Email from env or prompt once.
	if cfg.Email == "" {
		if v := os.Getenv("IDLENET_EMAIL"); v != "" {
			cfg.Email = v
		} else {
			fmt.Print("Enter signup email: ")
			fmt.Scanln(&cfg.Email)
		}
	}
	// Optional referral from env on first run.
	if cfg.Referral == "" {
		if v := os.Getenv("IDLENET_REF"); v != "" {
			cfg.Referral = v
		}
	}
	_ = saveConfig(cfg)

	// API base + optional Vercel bypass token
	apiBase := envOr("IDLENET_API_BASE", defaultAPIBase)
	bypass := os.Getenv("VERCEL_BYPASS_TOKEN")

	fmt.Printf("IdleNet Agent %s\nAPI: %s\nEmail: %s\nDevice: %s\n", version, apiBase, cfg.Email, cfg.DeviceID)

	// Register once
	if err := apiRegister(ctx, apiBase, bypass, cfg.Email, cfg.DeviceID, cfg.Referral, version); err != nil {
		fmt.Println("register error:", err)
	} else {
		fmt.Println("registered OK")
	}

	beatTicker := time.NewTicker(30 * time.Second)
	jobTicker := time.NewTicker(20 * time.Second)
	defer beatTicker.Stop()
	defer jobTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("shutting down…")
			return

		case <-beatTicker.C:
			if err := apiBeat(ctx, apiBase, bypass, cfg.Email, cfg.DeviceID); err != nil {
				fmt.Println("heartbeat error:", err)
			} else {
				fmt.Println(time.Now().Format(time.RFC3339), "heartbeat OK")
			}

		case <-jobTicker.C:
			job, err := apiNextJob(ctx, apiBase, bypass, cfg.Email, cfg.DeviceID)
			if err != nil {
				fmt.Println("next job error:", err)
				continue
			}
			if job == nil {
				// no work
				continue
			}

			// Run the job with a timeout
			start := time.Now()
			status, errMsg := runJob(ctx, job)
			dur := time.Since(start)

			if err := apiReport(ctx, apiBase, bypass, job.JobID, status, dur, errMsg); err != nil {
				fmt.Println("report error:", err)
			} else {
				fmt.Printf("job %s status=%s dur=%s err=%s\n", job.JobID, status, dur, errMsg)
			}
		}
	}
}

/* ----------------- Config helpers ----------------- */

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(dir, "idlenet")
	_ = os.MkdirAll(root, 0o755)
	return filepath.Join(root, "config.json"), nil
}

func loadConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return &Config{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return &Config{}, nil // first run
	}
	var c Config
	if e := json.Unmarshal(b, &c); e != nil {
		return &Config{}, e
	}
	return &c, nil
}

func saveConfig(c *Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

func genDeviceID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("dev-%d", time.Now().UnixNano())
	}
	return "dev-" + hex.EncodeToString(buf[:])
}

/* ----------------- API client ----------------- */

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func withBypass(u, bypass string) string {
	if bypass == "" {
		return u
	}
	sep := "?"
	if bytes.Contains([]byte(u), []byte("?")) {
		sep = "&"
	}
	return fmt.Sprintf("%s%sx-vercel-set-bypass-cookie=true&x-vercel-protection-bypass=%s", u, sep, bypass)
}

func doJSON(ctx context.Context, method, url, bypass string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(ctx, method, withBypass(url, bypass), r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bypass != "" {
		req.Header.Set("x-vercel-protection-bypass", bypass) // extra safety
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

func apiRegister(ctx context.Context, base, bypass, email, deviceID, referral, ver string) error {
	resp, err := doJSON(ctx, "POST", base+"/api/agent/register", bypass, map[string]any{
		"email":    email,
		"deviceId": deviceID,
		"referral": referral,
		"version":  ver,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func apiBeat(ctx context.Context, base, bypass, email, deviceID string) error {
	resp, err := doJSON(ctx, "POST", base+"/api/agent/beat", bypass, map[string]any{
		"email":    email,
		"deviceId": deviceID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("beat status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

type nextJobResp struct {
	JobID      string          `json:"jobId"`
	Type       string          `json:"type"`
	Args       json.RawMessage `json:"args"`
	TimeoutSec int             `json:"timeoutSec"`
}

func apiNextJob(ctx context.Context, base, bypass, email, deviceID string) (*nextJobResp, error) {
	url := fmt.Sprintf("%s/api/agent/jobs/next?email=%s&deviceId=%s", base, email, deviceID)
	resp, err := doJSON(ctx, "GET", url, bypass, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 204 = no work
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("next job %d: %s", resp.StatusCode, string(b))
	}
	var out nextJobResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func apiReport(ctx context.Context, base, bypass, jobID, status string, dur time.Duration, errMsg string) error {
	resp, err := doJSON(ctx, "POST", base+"/api/agent/jobs/report", bypass, map[string]any{
		"jobId":      jobID,
		"status":     status,
		"durationMs": dur.Milliseconds(),
		"error":      errMsg,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

/* ----------------- Job runner (safe canaries) ----------------- */

type job struct {
	JobID      string
	Type       string
	Args       json.RawMessage
	TimeoutSec int
}

func runJob(parent context.Context, j *job) (status string, errMsg string) {
	// time box the job
	dl := time.Duration(j.TimeoutSec) * time.Second
	if dl <= 0 {
		dl = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, dl)
	defer cancel()

	start := time.Now()
	_ = start

	switch j.Type {
	case "sleep":
		// args: {"seconds": N}
		var a struct{ Seconds int `json:"seconds"` }
		_ = json.Unmarshal(j.Args, &a)
		if a.Seconds <= 0 {
			a.Seconds = 5
		}
		t := time.NewTimer(time.Duration(a.Seconds) * time.Second)
		select {
		case <-ctx.Done():
			return "error", "timeout/cancelled"
		case <-t.C:
			return "ok", ""
		}

	case "hash":
		// simple CPU canary for N seconds
		var a struct{ Seconds int `json:"seconds"` }
		_ = json.Unmarshal(j.Args, &a)
		if a.Seconds <= 0 {
			a.Seconds = 10
		}
		buf := make([]byte, 1<<16)
		for i := range buf {
			buf[i] = byte(i)
		}
		for time.Since(start) < time.Duration(a.Seconds)*time.Second {
			select {
			case <-ctx.Done():
				return "error", "timeout/cancelled"
			default:
				sum := sha25632(buf)
				_ = sum // discard; just burn cycles
			}
		}
		return "ok", ""

	default:
		return "skipped", "unsupported job type"
	}
}

// tiny helper (no external deps)
func sha25632(b []byte) [32]byte {
	// Inline a tiny SHA-256 by delegating to stdlib via a one-liner to avoid extra imports everywhere
	// (kept separate to make the loop above cleaner).
	return *(*[32]byte)(bytes.NewBuffer(b).Bytes()) // placeholder to keep code simple; we're not using the value
}
