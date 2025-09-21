package ui

import (
    "embed"
    "encoding/json"
    "fmt"
    "net/http"
    "os/exec"
    "runtime"
    
    "github.com/ifruncillo/idlenet-agent/internal/config"
)

//go:embed settings.html
var settingsHTML string

// SettingsServer handles the web-based settings interface
type SettingsServer struct {
    cfg  *config.Config
    port int
}

// NewSettingsServer creates a new settings server
func NewSettingsServer(cfg *config.Config) *SettingsServer {
    return &SettingsServer{
        cfg:  cfg,
        port: 8765,
    }
}

// Start begins serving the settings interface
func (s *SettingsServer) Start() error {
    http.HandleFunc("/", s.handleIndex)
    http.HandleFunc("/api/config", s.handleConfig)
    http.HandleFunc("/api/save", s.handleSave)
    
    url := fmt.Sprintf("http://localhost:%d", s.port)
    
    // Open browser
    go func() {
        switch runtime.GOOS {
        case "windows":
            exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
        case "darwin":
            exec.Command("open", url).Start()
        case "linux":
            exec.Command("xdg-open", url).Start()
        }
    }()
    
    return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *SettingsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(settingsHTML))
}

func (s *SettingsServer) handleConfig(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(s.cfg)
}

func (s *SettingsServer) handleSave(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var updates config.Config
    if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Update configuration
    s.cfg.ResourceMode = updates.ResourceMode
    s.cfg.AllowBackground = updates.AllowBackground
    s.cfg.MaxCPUPercent = updates.MaxCPUPercent
    s.cfg.MaxMemoryMB = updates.MaxMemoryMB
    
    if err := config.Save(s.cfg); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}