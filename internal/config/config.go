package config

import (
    "time"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
)

type Config struct {
    Email      string `json:"email"`
    Referral   string `json:"referral,omitempty"`
    DeviceID   string `json:"device_id"`
    APIBase    string `json:"api_base"`
    Registered bool   `json:"registered"`
}

func configDir() (string, error) {
    switch runtime.GOOS {
    case "windows":
        programData := os.Getenv("ProgramData")
        if programData == "" {
            return "", fmt.Errorf("ProgramData not set")
        }
        return filepath.Join(programData, "IdleNet"), nil
    default:
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, ".idlenet"), nil
    }
}

func Load() (*Config, error) {
    dir, err := configDir()
    if err != nil {
        return nil, err
    }

    configPath := filepath.Join(dir, "config.json")
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            cfg := &Config{
                DeviceID: generateDeviceID(),
            }
            return cfg, nil
        }
        return nil, err
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    if cfg.DeviceID == "" {
        cfg.DeviceID = generateDeviceID()
    }

    return &cfg, nil
}

func Save(cfg *Config) error {
    dir, err := configDir()
    if err != nil {
        return err
    }

    configPath := filepath.Join(dir, "config.json")
    
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(configPath, data, 0644)
}

func generateDeviceID() string {
    // Simple ID generation - in production would use UUID
    return fmt.Sprintf("device-%d", time.Now().UnixNano())
}
