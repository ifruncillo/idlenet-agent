package resource

import (
    "runtime"
    "time"
    
    "github.com/ifruncillo/idlenet-agent/internal/idle"
)

// Manager controls how much system resources the agent can use
type Manager struct {
    userPreference   string
    lastCheck        time.Time
    currentCPULimit  int
    currentMemLimit  int
}

// NewManager creates a resource manager with user preferences
func NewManager(preference string) *Manager {
    if preference == "" {
        preference = "balanced"
    }
    
    return &Manager{
        userPreference: preference,
        lastCheck:      time.Now(),
    }
}

// GetLimits returns the current CPU and memory limits based on system activity
func (m *Manager) GetLimits() (cpuPercent, memPercent int) {
    // Cache results for 5 seconds
    if time.Since(m.lastCheck) < 5*time.Second {
        return m.currentCPULimit, m.currentMemLimit
    }
    
    m.lastCheck = time.Now()
    
    // Get current activity level
    activityLevel, err := idle.GetActivityLevel()
    if err != nil {
        // Conservative defaults if we can't determine activity
        m.currentCPULimit = 10
        m.currentMemLimit = 10
        return m.currentCPULimit, m.currentMemLimit
    }
    
    // Calculate limits based on preference and activity
    switch m.userPreference {
    case "aggressive":
        if activityLevel > 80 {
            m.currentCPULimit = 80
            m.currentMemLimit = 60
        } else if activityLevel > 50 {
            m.currentCPULimit = 50
            m.currentMemLimit = 40
        } else {
            m.currentCPULimit = 30
            m.currentMemLimit = 25
        }
        
    case "balanced":
        if activityLevel > 90 {
            m.currentCPULimit = 70
            m.currentMemLimit = 50
        } else if activityLevel > 60 {
            m.currentCPULimit = 40
            m.currentMemLimit = 30
        } else if activityLevel > 30 {
            m.currentCPULimit = 20
            m.currentMemLimit = 15
        } else {
            m.currentCPULimit = 10
            m.currentMemLimit = 10
        }
        
    case "conservative":
        if activityLevel > 95 {
            m.currentCPULimit = 50
            m.currentMemLimit = 30
        } else if activityLevel > 80 {
            m.currentCPULimit = 25
            m.currentMemLimit = 20
        } else {
            m.currentCPULimit = 5
            m.currentMemLimit = 5
        }
        
    case "idle-only":
        if activityLevel > 95 {
            m.currentCPULimit = 60
            m.currentMemLimit = 40
        } else {
            m.currentCPULimit = 0
            m.currentMemLimit = 0
        }
        
    default:
        m.currentCPULimit = 20
        m.currentMemLimit = 15
    }
    
    // Cap maximums for stability
    maxCPU := 80
    maxMem := 60
    
    if isLaptop() {
        maxCPU = 60
        maxMem = 40
    }
    
    if m.currentCPULimit > maxCPU {
        m.currentCPULimit = maxCPU
    }
    if m.currentMemLimit > maxMem {
        m.currentMemLimit = maxMem
    }
    
    return m.currentCPULimit, m.currentMemLimit
}

// ShouldRunJob determines if we should accept new jobs
func (m *Manager) ShouldRunJob() bool {
    cpu, _ := m.GetLimits()
    return cpu > 0
}

// GetCoreCount returns how many CPU cores we can use
func (m *Manager) GetCoreCount() int {
    totalCores := runtime.NumCPU()
    cpuLimit, _ := m.GetLimits()
    
    allowedCores := (totalCores * cpuLimit) / 100
    if allowedCores < 1 && cpuLimit > 0 {
        allowedCores = 1
    }
    
    return allowedCores
}

// isLaptop attempts to detect if running on a laptop
func isLaptop() bool {
    return runtime.NumCPU() <= 8
}