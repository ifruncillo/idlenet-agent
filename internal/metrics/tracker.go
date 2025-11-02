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
    mu            sync.RWMutex
    sessionStart  time.Time
    jobsCompleted int
    jobsFailed    int
    totalCPUTime  time.Duration
    totalEarnings float64
    currentMetrics *SystemMetrics
    metricsBuffer []*JobMetrics // Buffer for batched writes
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

    return &SystemMetrics{
        Timestamp:    time.Now(),
        CPUPercent:   t.currentMetrics.CPUPercent,
        MemoryMB:     t.currentMetrics.MemoryMB,
        JobsRunning:  t.currentMetrics.JobsRunning,
        TotalJobs:    t.jobsCompleted + t.jobsFailed,
        SessionHours: time.Since(t.sessionStart).Hours(),
        Earnings:     t.totalEarnings,
    }
}


func (t *Tracker) GetSessionStart() time.Time {
t.mu.RLock()
defer t.mu.RUnlock()
return t.sessionStart
}
func (t *Tracker) GetStats() (completed, failed int, cpuTime time.Duration, earnings float64) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    return t.jobsCompleted, t.jobsFailed, t.totalCPUTime, t.totalEarnings
}

func (t *Tracker) saveJobMetrics(job *JobMetrics) {
    // Buffer metrics writes to reduce disk I/O
    t.metricsBuffer = append(t.metricsBuffer, job)

    // Flush buffer when it reaches 10 jobs
    if len(t.metricsBuffer) >= 10 {
        t.flushMetricsBuffer()
    }
}

func (t *Tracker) flushMetricsBuffer() {
    if len(t.metricsBuffer) == 0 {
        return
    }

    homeDir, _ := os.UserHomeDir()
    metricsDir := filepath.Join(homeDir, ".idlenet", "metrics")
    os.MkdirAll(metricsDir, 0755)

    // Save to daily file
    filename := fmt.Sprintf("jobs_%s.json", time.Now().Format("2006-01-02"))
    filePath := filepath.Join(metricsDir, filename)

    file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return
    }
    defer file.Close()

    // Write all buffered metrics
    for _, job := range t.metricsBuffer {
        data, _ := json.Marshal(job)
        file.Write(data)
        file.WriteString("\n")
    }

    // Clear buffer
    t.metricsBuffer = t.metricsBuffer[:0]
}

// FlushMetrics forces a flush of buffered metrics (call on shutdown)
func (t *Tracker) FlushMetrics() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.flushMetricsBuffer()
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
