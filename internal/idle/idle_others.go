//go:build !windows

package idle

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "time"
)

// idleCache provides thread-safe caching for idle detection results
type idleCache struct {
    mu                sync.RWMutex
    lastCheckTime     time.Time
    lastIdleTime      time.Duration
    lastActivityLevel int
}

var cache = &idleCache{}

// GetIdleTime returns the system idle time for Linux and macOS
func GetIdleTime() (time.Duration, error) {
    // Check cache with read lock
    cache.mu.RLock()
    if time.Since(cache.lastCheckTime) < 5*time.Second {
        result := cache.lastIdleTime
        cache.mu.RUnlock()
        return result, nil
    }
    cache.mu.RUnlock()

    // Fetch new idle time
    var idleTime time.Duration
    var err error

    switch runtime.GOOS {
    case "linux":
        idleTime, err = getLinuxIdleTime()

    case "darwin":
        idleTime, err = getMacOSIdleTime()

    default:
        // Fallback for other Unix-like systems
        idleTime = 30 * time.Second
    }

    // Update cache with write lock (even on error to avoid repeated calls)
    cache.mu.Lock()
    cache.lastCheckTime = time.Now()
    if err == nil {
        cache.lastIdleTime = idleTime
    }
    cache.mu.Unlock()

    return idleTime, err
}

// getLinuxIdleTime attempts to get idle time on Linux systems
func getLinuxIdleTime() (time.Duration, error) {
    // Try xprintidle first (most accurate for X11)
    cmd := exec.Command("xprintidle")
    output, err := cmd.Output()
    if err == nil {
        milliseconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
        if err == nil {
            return time.Duration(milliseconds) * time.Millisecond, nil
        }
    }

    // Fallback: Check load average as a proxy for activity
    data, err := os.ReadFile("/proc/loadavg")
    if err != nil {
        return 0, fmt.Errorf("failed to read /proc/loadavg: %w", err)
    }

    parts := strings.Fields(string(data))
    if len(parts) < 1 {
        return 0, fmt.Errorf("invalid /proc/loadavg format")
    }

    load, err := strconv.ParseFloat(parts[0], 64)
    if err != nil {
        return 0, fmt.Errorf("failed to parse load average: %w", err)
    }

    // If load is low, system is likely idle
    // This is a heuristic: load < 0.5 suggests idle system
    if load < 0.5 {
        return 5 * time.Minute, nil // Estimate as idle
    } else if load < 1.0 {
        return 1 * time.Minute, nil // Moderate activity
    }
    return 10 * time.Second, nil // Active system
}

// getMacOSIdleTime attempts to get idle time on macOS
func getMacOSIdleTime() (time.Duration, error) {
    // Use ioreg to query idle time from IOHIDSystem
    cmd := exec.Command("ioreg", "-c", "IOHIDSystem")
    output, err := cmd.Output()
    if err != nil {
        // Fallback: return moderate idle time
        return 30 * time.Second, nil
    }

    // Parse output for HIDIdleTime (in nanoseconds)
    lines := strings.Split(string(output), "\n")
    for _, line := range lines {
        if strings.Contains(line, "HIDIdleTime") {
            // Extract the number from: "HIDIdleTime" = 12345678900
            parts := strings.Split(line, "=")
            if len(parts) == 2 {
                valueStr := strings.TrimSpace(parts[1])
                nanos, err := strconv.ParseInt(valueStr, 10, 64)
                if err == nil {
                    return time.Duration(nanos), nil
                }
            }
        }
    }

    // Fallback
    return 30 * time.Second, nil
}

// IsIdle returns true if the system has been idle for at least the specified duration
func IsIdle(duration time.Duration) (bool, error) {
    idleTime, err := GetIdleTime()
    if err != nil {
        return false, err
    }
    return idleTime >= duration, nil
}

// GetActivityLevel returns a percentage (0-100) representing how active the user is
func GetActivityLevel() (int, error) {
    // Check cache with read lock
    cache.mu.RLock()
    if time.Since(cache.lastCheckTime) < 5*time.Second {
        result := cache.lastActivityLevel
        cache.mu.RUnlock()
        return result, nil
    }
    cache.mu.RUnlock()

    // Get current idle time
    idleTime, err := GetIdleTime()
    if err != nil {
        // Update cache even on error to avoid repeated calls
        cache.mu.Lock()
        cache.lastCheckTime = time.Now()
        cache.lastActivityLevel = 50
        cache.mu.Unlock()
        return 50, nil // Default to moderate activity if we can't determine
    }

    // Convert idle time to activity level
    // 0 minutes idle = 100% active
    // 5+ minutes idle = 0% active
    idleMinutes := idleTime.Minutes()
    var activityLevel int
    if idleMinutes >= 5 {
        activityLevel = 0
    } else if idleMinutes >= 3 {
        activityLevel = 25
    } else if idleMinutes >= 1 {
        activityLevel = 50
    } else {
        activityLevel = 100
    }

    // Update cache
    cache.mu.Lock()
    cache.lastActivityLevel = activityLevel
    cache.mu.Unlock()

    return activityLevel, nil
}