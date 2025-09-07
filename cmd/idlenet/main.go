package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const version = "0.2.0"

// Default to your Vercel domain. You can override with env IDLENET_API.
var defaultAPIBase = "https://YOUR-VERCEL-DOMAIN" // <-- put your vercel domain here

type Config struct {
	Email    string `json:"email"`
	Referral string `json:"referral"`
	DeviceID string `json:"device_id"`
}

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
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return &Config{}, nil // first run
	}
	var c Config
	_ = json.Unmarshal(b, &c)
	return &c, nil
}

func saveConfig(c *Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, b, fs.FileMode(0o644))
}

func apiBase() string {
	if v := os.Getenv("IDLENET_API"); v != "" {
		return v
	}
	return defaultAPIBase
}

func postJSON(path string, payload any) error {
	url := apiBase() + path
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}

func main() {
	emailFlag := flag.String("email", "", "email used at signup")
	refFlag := flag.String("ref", "", "optional referral code")
	once := flag.Bool("once", false, "send one heartbeat then exit")
	flag.Parse()

	cfg, _ := loadConfig()
	if cfg.DeviceID == "" {
		cfg.DeviceID = uuid.NewString()
	}

	// Fill from flags if provided
	if *emailFlag != "" {
		cfg.Email = *emailFlag
	}
	if *refFlag != "" {
		cfg.Referral = *refFlag
	}

	// First-run prompt (very basic)
	if cfg.Email == "" {
		fmt.Print("Enter signup email: ")
		fmt.Scanln(&cfg.Email)
	}
	fmt.Printf("IdleNet Agent v%s\nEmail: %s\nDevice: %s\n", version, cfg.Email, cfg.DeviceID)

	// Save config and register
	_ = saveConfig(cfg)
	_ = postJSON("/api/agent/register", map[string]any{
		"email":    cfg.Email,
		"referral": cfg.Referral,
		"deviceId": cfg.DeviceID,
		"version":  version,
	})

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Optional: one-shot and exit (useful for testing)
	if *once {
		_ = postJSON("/api/agent/beat", map[string]any{
			"email":    cfg.Email,
			"deviceId": cfg.DeviceID,
		})
		fmt.Println("Heartbeat sent (once).")
		return
	}

	// Loop
	for {
		fmt.Println(time.Now().Format(time.RFC3339), "heartbeat")
		_ = postJSON("/api/agent/beat", map[string]any{
			"email":    cfg.Email,
			"deviceId": cfg.DeviceID,
		})
		<-ticker.C
	}
}
