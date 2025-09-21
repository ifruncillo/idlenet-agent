package updater

import (
    "syscall"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "time"
)

// SelfUpdater handles the self-replacement process
type SelfUpdater struct {
    currentExePath string
    backupPath     string
}

// NewSelfUpdater creates a new self-updater
func NewSelfUpdater() (*SelfUpdater, error) {
    exePath, err := os.Executable()
    if err != nil {
        return nil, err
    }
    
    return &SelfUpdater{
        currentExePath: exePath,
        backupPath:     exePath + ".backup",
    }, nil
}

// ApplyUpdate replaces the current executable with the new one
func (su *SelfUpdater) ApplyUpdate(newExePath string) error {
    // Step 1: Create backup of current executable
    if err := su.createBackup(); err != nil {
        return fmt.Errorf("failed to create backup: %w", err)
    }
    
    // Step 2: Replace executable
    if runtime.GOOS == "windows" {
        // Windows requires special handling
        return su.applyUpdateWindows(newExePath)
    }
    
    return su.applyUpdateUnix(newExePath)
}

// createBackup creates a backup of the current executable
func (su *SelfUpdater) createBackup() error {
    source, err := os.Open(su.currentExePath)
    if err != nil {
        return err
    }
    defer source.Close()
    
    backup, err := os.Create(su.backupPath)
    if err != nil {
        return err
    }
    defer backup.Close()
    
    _, err = io.Copy(backup, source)
    return err
}

// applyUpdateWindows handles Windows-specific update process
func (su *SelfUpdater) applyUpdateWindows(newExePath string) error {
    // Create a batch file that will:
    // 1. Wait for current process to exit
    // 2. Replace the executable
    // 3. Restart the agent
    // 4. Delete itself
    
    batchContent := fmt.Sprintf(`@echo off
echo Updating IdleNet Agent...
ping 127.0.0.1 -n 3 > nul
move /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, newExePath, su.currentExePath, su.currentExePath)
    
    batchPath := filepath.Join(os.TempDir(), "idlenet_update.bat")
    if err := os.WriteFile(batchPath, []byte(batchContent), 0755); err != nil {
        return err
    }
    
    // Execute the batch file
    cmd := exec.Command("cmd", "/c", batchPath)
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // Exit the current process
    fmt.Println("Update will be applied on restart...")
    time.Sleep(2 * time.Second)
    os.Exit(0)
    
    return nil
}

// applyUpdateUnix handles Unix-like systems update process
func (su *SelfUpdater) applyUpdateUnix(newExePath string) error {
    // Make new executable permission match current
    currentInfo, err := os.Stat(su.currentExePath)
    if err != nil {
        return err
    }
    
    if err := os.Chmod(newExePath, currentInfo.Mode()); err != nil {
        return err
    }
    
    // Replace the executable
    if err := os.Rename(newExePath, su.currentExePath); err != nil {
        return err
    }
    
    // Restart the process
    args := os.Args
    env := os.Environ()
    
    return syscall.Exec(su.currentExePath, args, env)
}

// Rollback restores the backup if update failed
func (su *SelfUpdater) Rollback() error {
    if _, err := os.Stat(su.backupPath); err != nil {
        return fmt.Errorf("no backup found")
    }
    
    return os.Rename(su.backupPath, su.currentExePath)
}

// CleanupBackup removes the backup file
func (su *SelfUpdater) CleanupBackup() error {
    return os.Remove(su.backupPath)
}