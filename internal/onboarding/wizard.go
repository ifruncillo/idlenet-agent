package onboarding

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    
    "github.com/ifruncillo/idlenet-agent/internal/config"
)

// SetupWizard guides new users through initial configuration
type SetupWizard struct {
    config *config.Config
}

// NewSetupWizard creates a new setup wizard
func NewSetupWizard() *SetupWizard {
    return &SetupWizard{}
}

// Run executes the setup wizard
func (w *SetupWizard) Run() (*config.Config, error) {
    fmt.Println("========================================")
    fmt.Println("   Welcome to IdleNet Agent Setup!")
    fmt.Println("========================================")
    fmt.Println()
    fmt.Println("Let's get you set up to start earning money")
    fmt.Println("from your computer's idle time.")
    fmt.Println()
    
    reader := bufio.NewReader(os.Stdin)
    
    // Get email
    fmt.Print("Enter your email address: ")
    email, _ := reader.ReadString('\n')
    email = strings.TrimSpace(email)
    
    // Get referral code if they have one
    fmt.Print("Referral code (press Enter to skip): ")
    referral, _ := reader.ReadString('\n')
    referral = strings.TrimSpace(referral)
    
    // Choose resource mode
    fmt.Println()
    fmt.Println("How should IdleNet use your computer?")
    fmt.Println("1. Aggressive - Maximum earnings (uses more resources)")
    fmt.Println("2. Balanced - Good earnings with moderate resource use")
    fmt.Println("3. Conservative - Minimal impact on your system")
    fmt.Println("4. Idle Only - Only run when you're not using the computer")
    fmt.Print("Choose (1-4) [default: 2]: ")
    
    choice, _ := reader.ReadString('\n')
    choice = strings.TrimSpace(choice)
    
    resourceMode := "balanced"
    switch choice {
    case "1":
        resourceMode = "aggressive"
    case "3":
        resourceMode = "conservative"
    case "4":
        resourceMode = "idle-only"
    }
    
    // Ask about startup
    fmt.Print("Start IdleNet automatically when Windows starts? (y/n) [y]: ")
    autostart, _ := reader.ReadString('\n')
    autostart = strings.TrimSpace(strings.ToLower(autostart))
    
    enableAutostart := autostart != "n" && autostart != "no"
    
    // Create configuration
    cfg := &config.Config{
        Email:        email,
        Referral:     referral,
        ResourceMode: resourceMode,
        APIBase:      "https://idlenet-pilot-qi7t.vercel.app",
    }
    
    // Save configuration
    if err := config.Save(cfg); err != nil {
        return nil, err
    }
    
    // Set up autostart if requested
    if enableAutostart {
        w.enableAutostart()
    }
    
    fmt.Println()
    fmt.Println("========================================")
    fmt.Println("Setup complete! IdleNet Agent is ready.")
    fmt.Println()
    fmt.Printf("You're earning in %s mode.\n", resourceMode)
    fmt.Println("The agent will run in the background.")
    fmt.Println()
    fmt.Println("Check your earnings at:")
    fmt.Println("https://idlenet-pilot-qi7t.vercel.app")
    fmt.Println("========================================")
    
    return cfg, nil
}

// enableAutostart adds the agent to Windows startup
func (w *SetupWizard) enableAutostart() error {
    // This would use Windows registry to add to startup
    // Implementation depends on platform
    return nil
}

// IsFirstRun checks if this is the first time running
func IsFirstRun() bool {
    cfg, err := config.Load()
    if err != nil {
        return true
    }
    return cfg.Email == ""
}