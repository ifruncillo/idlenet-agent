package dashboard

import (
"context"
"encoding/json"
"fmt"
"net/http"
"os"
"os/exec"
"path/filepath"
"runtime"
"time"
"github.com/ifruncillo/idlenet-agent/internal/config"
"github.com/ifruncillo/idlenet-agent/internal/metrics"
)

type Server struct {
cfg     *config.Config
metrics *metrics.Tracker
history *History
port    int
}

func New(cfg *config.Config, metricsTracker *metrics.Tracker) *Server {
return &Server{
cfg:     cfg,
metrics: metricsTracker,
history: NewHistory(),
port:    8090,
}
}

func (s *Server) Start(ctx context.Context) error {
mux := http.NewServeMux()

cwd, _ := os.Getwd()
assetsPath := filepath.Join(cwd, "internal", "dashboard", "assets")
fmt.Printf("Serving files from: %s\n", assetsPath)

fs := http.FileServer(http.Dir(assetsPath))
mux.Handle("/", fs)
mux.HandleFunc("/api/stats", s.handleStats)
mux.HandleFunc("/api/history", s.handleHistory)

srv := &http.Server{Addr: fmt.Sprintf(":%d", s.port), Handler: mux}

// Record data point every minute
go func() {
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()
for {
select {
case <-ticker.C:
completed, _, _, earnings := s.metrics.GetStats()
s.history.Add(earnings, completed)
case <-ctx.Done():
return
}
}
}()

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

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
duration := r.URL.Query().Get("range")
if duration == "" {
duration = "24h"
}

points := s.history.GetRange(duration)
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(points)
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
