package dashboard

import (
"encoding/json"
"os"
"path/filepath"
"sync"
"time"
)

type DataPoint struct {
Time     time.Time `json:"time"`
Earnings float64   `json:"earnings"`
Jobs     int       `json:"jobs"`
}

type History struct {
mu     sync.Mutex
Points []DataPoint `json:"points"`
file   string
}

func NewHistory() *History {
dir, _ := os.UserConfigDir()
file := filepath.Join(dir, "idlenet", "history.json")

h := &History{
file:   file,
Points: []DataPoint{},
}
h.load()
return h
}

func (h *History) load() {
data, err := os.ReadFile(h.file)
if err == nil {
json.Unmarshal(data, &h.Points)
}
}

func (h *History) save() {
data, _ := json.MarshalIndent(h.Points, "", "  ")
os.WriteFile(h.file, data, 0644)
}

func (h *History) Add(earnings float64, jobs int) {
h.mu.Lock()
defer h.mu.Unlock()

h.Points = append(h.Points, DataPoint{
Time:     time.Now(),
Earnings: earnings,
Jobs:     jobs,
})

// Keep only last 30 days
cutoff := time.Now().AddDate(0, 0, -30)
var filtered []DataPoint
for _, p := range h.Points {
if p.Time.After(cutoff) {
filtered = append(filtered, p)
}
}
h.Points = filtered
h.save()
}

func (h *History) GetRange(duration string) []DataPoint {
h.mu.Lock()
defer h.mu.Unlock()

var cutoff time.Time
now := time.Now()

switch duration {
case "1h":
cutoff = now.Add(-1 * time.Hour)
case "24h":
cutoff = now.Add(-24 * time.Hour)
case "7d":
cutoff = now.AddDate(0, 0, -7)
case "30d":
cutoff = now.AddDate(0, 0, -30)
default:
cutoff = now.Add(-24 * time.Hour)
}

var result []DataPoint
for _, p := range h.Points {
if p.Time.After(cutoff) {
result = append(result, p)
}
}
return result
}
