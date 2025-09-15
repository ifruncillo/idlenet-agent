package updater

import (
    "fmt"
    "time"
)

// UpdateManager coordinates the entire update process
type UpdateManager struct {
    versionChecker *VersionChecker
    downloader     *Downloader
    selfUpdater    *SelfUpdater
    currentVersion string
}

// NewUpdateManager creates a new update manager
func NewUpdateManager(currentVersion string) (*UpdateManager, error) {
    downloader, err := NewDownloader()
    if err != nil {
        return nil, err
    }
    
    selfUpdater, err := NewSelfUpdater()
    if err != nil {
        return nil, err
    }
    
    return &UpdateManager{
        versionChecker: NewVersionChecker(currentVersion),
        downloader:     downloader,
        selfUpdater:    selfUpdater,
        currentVersion: currentVersion,
    }, nil
}

// CheckAndUpdate checks for updates and applies them if available
func (um *UpdateManager) CheckAndUpdate(autoApply bool) error {
    fmt.Println("Checking for updates...")
    
    release, hasUpdate, err := um.versionChecker.CheckForUpdate()
    if err != nil {
        return fmt.Errorf("failed to check for updates: %w", err)
    }
    
    if !hasUpdate {
        fmt.Println("You're running the latest version")
        return nil
    }
    
    fmt.Printf("New version available: %s (current: %s)\n", 
        release.TagName, um.currentVersion)
    
    if !autoApply {
        fmt.Println("Run with --update flag to apply update")
        return nil
    }
    
    // Download the update
    fmt.Println("Downloading update...")
    updatePath, err := um.downloader.DownloadUpdate(release)
    if err != nil {
        return fmt.Errorf("failed to download update: %w", err)
    }
    
    // Apply the update
    fmt.Println("Applying update...")
    if err := um.selfUpdater.ApplyUpdate(updatePath); err != nil {
        // Try to rollback on failure
        um.selfUpdater.Rollback()
        return fmt.Errorf("failed to apply update: %w", err)
    }
    
    return nil
}

// BackgroundUpdateCheck runs periodic update checks
func (um *UpdateManager) BackgroundUpdateCheck(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        release, hasUpdate, err := um.versionChecker.CheckForUpdate()
        if err != nil {
            continue
        }
        
        if hasUpdate {
            fmt.Printf("\n[Update Available] Version %s is ready to install\n", 
                release.TagName)
            fmt.Println("Restart the agent to apply the update")
        }
    }
}