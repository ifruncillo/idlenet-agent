package metrics

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type Tracker struct {
    mu           sync.RWMutex
    sessionStart time.Time
    jobsCompleted int
    jobsFailed    int
    totalCPUTime  time.Duration
    totalEarnings float64
    currentMetrics *SystemMetrics
}

type SystemMetrics struct {
    Timestamp    time.Time `json:"timestamp"`
    CPUPercent   float64   `json:"cpu_percent"`
    MemoryMB     int       `json:"memory_mb"`
    JobsRunning  int       `json:"jobs_running"`
    TotalJobs    int       `json:"total_jobs"`
    SessionHours float64   `json:"session_hours"`
    Earnings     float64   `json:"earnings"`
}

type JobMetrics struct {
    JobID        string    `json:"job_id"`
    DeviceID     string    `json:"device_id"`
    StartTime    time.Time `json:"start_time"`
    EndTime      time.Time `json:"end_time"`
    CPUSeconds   float64   `json:"cpu_seconds"`
    MemoryMB     int       `json:"memory_mb"`
    Success      bool      `json:"success"`
    ErrorMessage string    `json:"error_message,omitempty"`
    Earnings     float64   `json:"earnings"`
}

func NewTracker() *Tracker {
    return &Tracker{
        sessionStart: time.Now(),
        currentMetrics: &SystemMetrics{
            Timestamp: time.Now(),
        },
    }
}

func (t *Tracker) RecordJobStart(jobID string) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if t.currentMetrics.JobsRunning < 0 {
        t.currentMetrics.JobsRunning = 0
    }
    t.currentMetrics.JobsRunning++
}

func (t *Tracker) RecordJobComplete(job *JobMetrics) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if job.Success {
        t.jobsCompleted++
    } else {
        t.jobsFailed++
    }
    
    duration := job.EndTime.Sub(job.StartTime)
    t.totalCPUTime += duration
    
    // Calculate earnings: $0.001 per CPU second
    earnings := duration.Seconds() * 0.001
    job.Earnings = earnings
    t.totalEarnings += earnings
    
    if t.currentMetrics.JobsRunning > 0 {
        t.currentMetrics.JobsRunning--
    }
    
    t.currentMetrics.TotalJobs = t.jobsCompleted + t.jobsFailed
    t.currentMetrics.Earnings = t.totalEarnings
    
    // Save job metrics to file
    t.saveJobMetrics(job)
}

func (t *Tracker) GetCurrentMetrics() *SystemMetrics {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    metrics := *t.currentMetrics
    metrics.Timestamp = time.Now()
    metrics.SessionHours = time.Since(t.sessionStart).Hours()
    metrics.Earnings = t.totalEarnings
    
    return &metrics
}

func (t *Tracker) GetStats() (completed, failed int, cpuTime time.Duration, earnings float64) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    return t.jobsCompleted, t.jobsFailed, t.totalCPUTime, t.totalEarnings
}

func (t *Tracker) saveJobMetrics(job *JobMetrics) {
    homeDir, _ := os.UserHomeDir()
    metricsDir := filepath.Join(homeDir, ".idlenet", "metrics")
    os.MkdirAll(metricsDir, 0755)
    
    // Save to daily file
    filename := fmt.Sprintf("jobs_%s.json", time.Now().Format("2006-01-02"))
    filepath := filepath.Join(metricsDir, filename)
    
    file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return
    }
    defer file.Close()
    
    data, _ := json.Marshal(job)
    file.Write(data)
    file.WriteString("\n")
}

func CalculateEarnings(cpuSeconds float64, memoryMB int) float64 {
    // Base rate: $0.001 per CPU second
    baseRate := cpuSeconds * 0.001
    
    // Memory bonus: +10% for high memory jobs
    memoryBonus := 0.0
    if memoryMB > 1024 {
        memoryBonus = baseRate * 0.1
    }
    
    return baseRate + memoryBonus
}