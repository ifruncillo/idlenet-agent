package ui

import (
    "fmt"
    "os"
    "time"
    
    "github.com/getlantern/systray"
    "github.com/ifruncillo/idlenet-agent/internal/config"
)

// TrayApp manages the system tray interface
type TrayApp struct {
    cfg           *config.Config
    statusItem    *systray.MenuItem
    earningsItem  *systray.MenuItem
    settingsItem  *systray.MenuItem
    quitItem      *systray.MenuItem
    
    isRunning     bool
    totalRuntime  time.Duration
    sessionStart  time.Time
}

// Start initializes the system tray application
func Start(cfg *config.Config) {
    app := &TrayApp{
        cfg:          cfg,
        sessionStart: time.Now(),
    }
    
    systray.Run(app.onReady, app.onExit)
}

func (app *TrayApp) onReady() {
    // Set icon and tooltip
    systray.SetIcon(getIcon())
    systray.SetTitle("IdleNet Agent")
    systray.SetTooltip("IdleNet - Earning while idle")
    
    // Create menu items
    app.statusItem = systray.AddMenuItem("Status: Running", "Agent status")
    app.earningsItem = systray.AddMenuItem("Session: $0.00", "Earnings this session")
    systray.AddSeparator()
    
    // Resource mode submenu
    modeMenu := systray.AddMenuItem("Resource Mode", "Change resource usage")
    modeAggressive := modeMenu.AddSubMenuItem("Aggressive", "Maximum earnings")
    modeBalanced := modeMenu.AddSubMenuItem("Balanced", "Default")
    modeConservative := modeMenu.AddSubMenuItem("Conservative", "Minimal impact")
    modeIdleOnly := modeMenu.AddSubMenuItem("Idle Only", "Only when idle")
    
    // Set checkmark on current mode
    switch app.cfg.ResourceMode {
    case "aggressive":
        modeAggressive.Check()
    case "conservative":
        modeConservative.Check()
    case "idle-only":
        modeIdleOnly.Check()
    default:
        modeBalanced.Check()
    }
    
    systray.AddSeparator()
    app.settingsItem = systray.AddMenuItem("Settings", "Open settings")
    systray.AddMenuItem("View Dashboard", "Open earnings dashboard")
    systray.AddSeparator()
    app.quitItem = systray.AddMenuItem("Quit", "Stop IdleNet Agent")
    
    // Handle menu clicks
    go app.handleMenuClicks(modeAggressive, modeBalanced, modeConservative, modeIdleOnly)
    
    // Update status periodically
    go app.updateStatus()
}

func (app *TrayApp) handleMenuClicks(aggressive, balanced, conservative, idleOnly *systray.MenuItem) {
    for {
        select {
        case <-aggressive.ClickedCh:
            app.setResourceMode("aggressive", aggressive, balanced, conservative, idleOnly)
            
        case <-balanced.ClickedCh:
            app.setResourceMode("balanced", aggressive, balanced, conservative, idleOnly)
            
        case <-conservative.ClickedCh:
            app.setResourceMode("conservative", aggressive, balanced, conservative, idleOnly)
            
        case <-idleOnly.ClickedCh:
            app.setResourceMode("idle-only", aggressive, balanced, conservative, idleOnly)
            
        case <-app.settingsItem.ClickedCh:
            // TODO: Open settings window
            fmt.Println("Settings clicked")
            
        case <-app.quitItem.ClickedCh:
            systray.Quit()
        }
    }
}

func (app *TrayApp) setResourceMode(mode string, items ...*systray.MenuItem) {
    // Update config
    app.cfg.ResourceMode = mode
    config.Save(app.cfg)
    
    // Update checkmarks
    for _, item := range items {
        item.Uncheck()
    }
    
    switch mode {
    case "aggressive":
        items[0].Check()
    case "balanced":
        items[1].Check()
    case "conservative":
        items[2].Check()
    case "idle-only":
        items[3].Check()
    }
    
    // Show notification
    app.statusItem.SetTitle(fmt.Sprintf("Mode changed to: %s", mode))
}

func (app *TrayApp) updateStatus() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        runtime := time.Since(app.sessionStart)
        hours := runtime.Hours()
        
        // Calculate simulated earnings (replace with real calculation)
        // Assuming $0.10 per hour for demo
        earnings := hours * 0.10
        
        app.earningsItem.SetTitle(fmt.Sprintf("Session: $%.2f (%.1f hrs)", earnings, hours))
        
        if app.isRunning {
            app.statusItem.SetTitle("Status: Running ✓")
        } else {
            app.statusItem.SetTitle("Status: Idle")
        }
    }
}

func (app *TrayApp) onExit() {
    // Cleanup
}

// getIcon returns the icon data (you'll need to embed an actual icon)
func getIcon() []byte {
    // This is a placeholder - you should embed an actual .ico file
    // For now, returning empty will use system default
    return []byte{}
}