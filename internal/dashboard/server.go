package dashboard

import (
"context"
"encoding/json"
"fmt"
"net/http"
"os/exec"
"runtime"
"time"
"path/filepath"
"os"
"github.com/ifruncillo/idlenet-agent/internal/config"
"github.com/ifruncillo/idlenet-agent/internal/metrics"
)

type Server struct {
cfg     *config.Config
metrics *metrics.Tracker
port    int
}

func New(cfg *config.Config, metricsTracker *metrics.Tracker) *Server {
return &Server{cfg: cfg, metrics: metricsTracker, port: 8090}
}

func (s *Server) Start(ctx context.Context) error {
mux := http.NewServeMux()

// Get current working directory
cwd, _ := os.Getwd()
assetsPath := filepath.Join(cwd, "internal", "dashboard", "assets")
fmt.Printf("Serving files from: %s\n", assetsPath)

// Serve static files
fs := http.FileServer(http.Dir(assetsPath))
mux.Handle("/", fs)

// API endpoint
mux.HandleFunc("/api/stats", s.handleStats)

srv := &http.Server{Addr: fmt.Sprintf(":%d", s.port), Handler: mux}

go func() {
time.Sleep(2 * time.Second)
s.openBrowser(fmt.Sprintf("http://localhost:%d", s.port))
}()

go func() {
<-ctx.Done()
srv.Shutdown(context.Background())
}()

return srv.ListenAndServe()
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
completed, failed, cpuTime, earnings := s.metrics.GetStats()
stats := map[string]any{
"deviceId": s.cfg.DeviceID,
"email": s.cfg.Email,
"completed": completed,
"failed": failed,
"earnings": earnings,
"cpuTime": cpuTime.Seconds(),
"uptime": time.Since(s.metrics.GetSessionStart()).Seconds(),
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(stats)
}

func (s *Server) openBrowser(url string) {
var cmd *exec.Cmd
switch runtime.GOOS {
case "windows":
cmd = exec.Command("cmd", "/c", "start", url)
case "darwin":
cmd = exec.Command("open", url)
default:
cmd = exec.Command("xdg-open", url)
}
cmd.Start()
}
