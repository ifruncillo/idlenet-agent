package dashboard

import (
"encoding/json"
"fmt"
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
dirty  bool // Track if save is needed
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
if err != nil {
if !os.IsNotExist(err) {
fmt.Printf("Warning: failed to read history file: %v\n", err)
}
return
}
if err := json.Unmarshal(data, &h.Points); err != nil {
fmt.Printf("Warning: corrupted history file, starting fresh: %v\n", err)
h.Points = []DataPoint{}
}
}

func (h *History) save() {
// Ensure directory exists
dir := filepath.Dir(h.file)
if err := os.MkdirAll(dir, 0755); err != nil {
fmt.Printf("Warning: failed to create history directory: %v\n", err)
return
}

data, err := json.MarshalIndent(h.Points, "", "  ")
if err != nil {
fmt.Printf("Warning: failed to marshal history: %v\n", err)
return
}

if err := os.WriteFile(h.file, data, 0644); err != nil {
fmt.Printf("Warning: failed to write history file: %v\n", err)
}
}

func (h *History) Add(earnings float64, jobs int) {
h.mu.Lock()
defer h.mu.Unlock()

h.Points = append(h.Points, DataPoint{
Time:     time.Now(),
Earnings: earnings,
Jobs:     jobs,
})
h.dirty = true

// Clean old data only once per hour to reduce overhead
if len(h.Points) > 0 && len(h.Points)%60 == 0 {
h.cleanOldData()
}
}

// cleanOldData removes data points older than 30 days (should be called with lock held)
func (h *History) cleanOldData() {
cutoff := time.Now().AddDate(0, 0, -30)

// In-place filtering for better performance
n := 0
for i := range h.Points {
if h.Points[i].Time.After(cutoff) {
h.Points[n] = h.Points[i]
n++
}
}
h.Points = h.Points[:n]
}

// Flush writes dirty data to disk (call periodically)
func (h *History) Flush() {
h.mu.Lock()
defer h.mu.Unlock()

if h.dirty {
h.save()
h.dirty = false
}
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
