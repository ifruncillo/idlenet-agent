package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "time"
    "crypto/rand"
    "encoding/hex"
)

// Config holds all persistent settings for the agent
type Config struct {
    Email             string    `json:"email"`
    DeviceID          string    `json:"device_id"`
    Referral          string    `json:"referral,omitempty"`
    APIBase           string    `json:"api_base"`
    Registered        bool      `json:"registered"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
    
    // User preferences for resource usage
    ResourceMode      string    `json:"resource_mode"`      // aggressive, balanced, conservative, idle-only
    AllowBackground   bool      `json:"allow_background"`   // Run jobs while system is in use
    MaxCPUPercent     int       `json:"max_cpu_percent"`    // Override max CPU usage
    MaxMemoryMB       int       `json:"max_memory_mb"`      // Override max memory usage
}

// Existing functions remain the same...
func configDir() (string, error) {
    switch runtime.GOOS {
    case "windows":
        appData := os.Getenv("APPDATA")
        if appData == "" {
            return "", fmt.Errorf("APPDATA environment variable not set")
        }
        return filepath.Join(appData, "IdleNet"), nil
    case "darwin":
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, "Library", "Application Support", "IdleNet"), nil
    default:
        home, err := os.UserHomeDir()
        if err != nil {
            return "", err
        }
        return filepath.Join(home, ".config", "idlenet"), nil
    }
}

func Load() (*Config, error) {
    dir, err := configDir()
    if err != nil {
        return nil, fmt.Errorf("failed to get config directory: %w", err)
    }
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create config directory: %w", err)
    }
    
    configPath := filepath.Join(dir, "config.json")
    
    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            cfg := &Config{
                DeviceID:        generateDeviceID(),
                APIBase:         "https://idlenet-pilot-qi7t.vercel.app",
                CreatedAt:       time.Now(),
                UpdatedAt:       time.Now(),
                ResourceMode:    "balanced",
                AllowBackground: false,
            }
            return cfg, nil
        }
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    if cfg.DeviceID == "" {
        cfg.DeviceID = generateDeviceID()
        cfg.UpdatedAt = time.Now()
    }
    
    if cfg.APIBase == "" {
        cfg.APIBase = "https://idlenet-pilot-qi7t.vercel.app"
    }
    
    if cfg.ResourceMode == "" {
        cfg.ResourceMode = "balanced"
    }
    
    return &cfg, nil
}

func Save(cfg *Config) error {
    dir, err := configDir()
    if err != nil {
        return fmt.Errorf("failed to get config directory: %w", err)
    }
    
    configPath := filepath.Join(dir, "config.json")
    cfg.UpdatedAt = time.Now()
    
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }
    
    tempPath := configPath + ".tmp"
    if err := os.WriteFile(tempPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }
    
    if err := os.Rename(tempPath, configPath); err != nil {
        os.Remove(tempPath)
        if err := os.WriteFile(configPath, data, 0644); err != nil {
            return fmt.Errorf("failed to save config: %w", err)
        }
    }
    
    return nil
}

func generateDeviceID() string {
    var bytes [16]byte
    if _, err := rand.Read(bytes[:]); err != nil {
        return fmt.Sprintf("device-%d", time.Now().UnixNano())
    }
    return "device-" + hex.EncodeToString(bytes[:])
}

func ConfigPath() (string, error) {
    dir, err := configDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "config.json"), nil
}