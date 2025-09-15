//go:build !windows

package idle

import (
    "time"
)

// GetIdleTime returns a simulated idle time for non-Windows platforms
// TODO: Implement actual idle detection for macOS and Linux
func GetIdleTime() (time.Duration, error) {
    // For now, return a default value
    // This ensures the agent still compiles and runs on all platforms
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
    // Simplified implementation for non-Windows
    return 50, nil
}