//go:build windows

package idle

import (
    "syscall"
    "time"
    "unsafe"
)

var (
    user32                  = syscall.NewLazyDLL("user32.dll")
    kernel32                = syscall.NewLazyDLL("kernel32.dll")
    procGetLastInputInfo    = user32.NewProc("GetLastInputInfo")
    procGetTickCount        = kernel32.NewProc("GetTickCount")
)

type lastInputInfo struct {
    cbSize uint32
    dwTime uint32
}

// GetIdleTime returns how long the system has been idle
func GetIdleTime() (time.Duration, error) {
    var info lastInputInfo
    info.cbSize = uint32(unsafe.Sizeof(info))
    
    ret, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
    if ret == 0 {
        return 0, err
    }
    
    tick, _, _ := procGetTickCount.Call()
    
    idleMillis := uint32(tick) - info.dwTime
    return time.Duration(idleMillis) * time.Millisecond, nil
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
// 0 = very active, 100 = completely idle
func GetActivityLevel() (int, error) {
    idleTime, err := GetIdleTime()
    if err != nil {
        return 0, err
    }
    
    // Scale idle time to activity level
    // < 1 second = 0% (very active)
    // > 5 minutes = 100% (completely idle)
    if idleTime < time.Second {
        return 0, nil
    }
    if idleTime > 5*time.Minute {
        return 100, nil
    }
    
    // Linear scale between 1 second and 5 minutes
    seconds := int(idleTime.Seconds())
    maxSeconds := 300 // 5 minutes
    level := (seconds * 100) / maxSeconds
    
    return level, nil
}