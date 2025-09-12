package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Email      string    `json:"email,omitempty"`
	Referral   string    `json:"referral,omitempty"`
	DeviceID   string    `json:"deviceId"`
	Registered bool      `json:"registered,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func fileDir() (string, error) {
	if runtime.GOOS == "windows" {
		progData := os.Getenv("ProgramData")
		if progData == "" {
			return "", errors.New("%ProgramData% not set")
		}
		return filepath.Join(progData, "IdleNet"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".idlenet"), nil
}

func FilePath() (string, error) {
	dir, err := fileDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func ensureDir() (string, error) {
	dir, err := fileDir()
	if err != nil {
		return "", err
	}
	// 0700 on unix; Windows ignores perms.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func Load() (*Config, error) {
	path, err := FilePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// First-run: create default with new device ID.
			c := &Config{
				DeviceID:  uuid.NewString(),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := Save(c); err != nil {
				return nil, err
			}
			return c, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	// Backfill fields if needed (e.g., DeviceID missing)
	changed := false
	if c.DeviceID == "" {
		c.DeviceID = uuid.NewString()
		c.CreatedAt = time.Now().UTC()
		changed = true
	}
	if changed {
		_ = Save(&c)
	}
	return &c, nil
}

func Save(c *Config) error {
	if _, err := ensureDir(); err != nil {
		return err
	}
	path, err := FilePath()
	if err != nil {
		return err
	}
	c.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// 0600 on unix; Windows ignores perms.
	return os.WriteFile(path, b, 0o600)
}
