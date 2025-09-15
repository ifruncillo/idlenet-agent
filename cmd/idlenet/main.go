package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/ifruncillo/idlenet-agent/internal/api"
    "github.com/ifruncillo/idlenet-agent/internal/config"
)

const version = "v1.0.0"

func main() {
    fmt.Printf("IdleNet Agent %s\n", version)
    fmt.Println("========================================")
    
    // Load or create configuration
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("Failed to load configuration: %v\n", err)
        os.Exit(1)
    }
    
    // Check if we need to get the email from the user
    if cfg.Email == "" {
        if email := os.Getenv("IDLENET_EMAIL"); email != "" {
            cfg.Email = email
            fmt.Printf("Using email from environment: %s\n", email)
        } else {
            fmt.Print("Enter your email address: ")
            fmt.Scanln(&cfg.Email)
        }
        
        if err := config.Save(cfg); err != nil {
            fmt.Printf("Warning: Failed to save config: %v\n", err)
        }
    }
    
    // Get optional referral code from environment (only on first run)
    if cfg.Referral == "" {
        if ref := os.Getenv("IDLENET_REF"); ref != "" {
            cfg.Referral = ref
            fmt.Printf("Using referral code: %s\n", ref)
            config.Save(cfg)
        }
    }
    
    // Display configuration
    fmt.Printf("Email: %s\n", cfg.Email)
    fmt.Printf("Device ID: %s\n", cfg.DeviceID)
    fmt.Printf("API Base: %s\n", cfg.APIBase)
    
    // Create API client
    apiClient := api.NewClient(cfg.APIBase, cfg.Email, cfg.DeviceID)
    
    // Check for Vercel bypass token (for protected deployments)
    if bypass := os.Getenv("VERCEL_BYPASS_TOKEN"); bypass != "" {
        apiClient.SetBypassToken(bypass)
        fmt.Println("Using Vercel bypass token for protected deployment")
    }
    
    // Register with the server if we haven't already
    if !cfg.Registered {
        fmt.Print("Registering with server... ")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        err := apiClient.Register(ctx, cfg.Referral, version)
        cancel()
        
        if err != nil {
            fmt.Printf("Failed: %v\n", err)
            fmt.Println("Will retry during heartbeat")
        } else {
            fmt.Println("Success!")
            cfg.Registered = true
            config.Save(cfg)
        }
    } else {
        fmt.Println("Status: Already registered")
    }
    
    fmt.Println("========================================")
    
    // Set up signal handling for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Start heartbeat ticker
    heartbeatTicker := time.NewTicker(30 * time.Second)
    defer heartbeatTicker.Stop()
    
    // Job check ticker (every 20 seconds, but only works when idle)
    jobTicker := time.NewTicker(20 * time.Second)
    defer jobTicker.Stop()
    
    fmt.Println("Agent running. Press Ctrl+C to stop.")
    fmt.Printf("[%s] Starting heartbeat system...\n", time.Now().Format("15:04:05"))
    
    // Main event loop
    for {
        select {
        case <-ctx.Done():
            fmt.Println("\nShutting down gracefully...")
            return
            
        case <-sigChan:
            fmt.Println("\nReceived shutdown signal")
            cancel()
            
        case <-heartbeatTicker.C:
            timestamp := time.Now().Format("15:04:05")
            
            // Send heartbeat to server
            beatCtx, beatCancel := context.WithTimeout(ctx, 5*time.Second)
            err := apiClient.Beat(beatCtx)
            beatCancel()
            
            if err != nil {
                fmt.Printf("[%s] Heartbeat failed: %v\n", timestamp, err)
                
                // If we're not registered and heartbeat failed, try registering again
                if !cfg.Registered {
                    regCtx, regCancel := context.WithTimeout(ctx, 10*time.Second)
                    if err := apiClient.Register(regCtx, cfg.Referral, version); err == nil {
                        fmt.Printf("[%s] Successfully registered with server\n", timestamp)
                        cfg.Registered = true
                        config.Save(cfg)
                    }
                    regCancel()
                }
            } else {
                fmt.Printf("[%s] Heartbeat sent successfully\n", timestamp)
            }
            
        case <-jobTicker.C:
            // TODO: Check for jobs (only when system is idle)
            // For now, just log that we checked
            timestamp := time.Now().Format("15:04:05")
            
            jobCtx, jobCancel := context.WithTimeout(ctx, 5*time.Second)
            job, err := apiClient.GetNextJob(jobCtx)
            jobCancel()
            
            if err != nil {
                fmt.Printf("[%s] Failed to check for jobs: %v\n", timestamp, err)
            } else if job != nil {
                fmt.Printf("[%s] Received job %s (type: %s)\n", timestamp, job.ID, job.Type)
                // TODO: Execute the job
            }
            // If job is nil and err is nil, there's simply no work available (normal)
        }
    }
}