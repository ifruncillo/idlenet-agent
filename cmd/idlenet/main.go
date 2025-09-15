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
    "github.com/ifruncillo/idlenet-agent/internal/executor"
    "github.com/ifruncillo/idlenet-agent/internal/idle"
    "github.com/ifruncillo/idlenet-agent/internal/resource"
)

const version = "v1.0.0"

func main() {
    fmt.Printf("IdleNet Agent %s\n", version)
    fmt.Println("========================================")
    
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("Failed to load configuration: %v\n", err)
        os.Exit(1)
    }
    
    // Get email if not set
    if cfg.Email == "" {
        if email := os.Getenv("IDLENET_EMAIL"); email != "" {
            cfg.Email = email
        } else {
            fmt.Print("Enter your email address: ")
            fmt.Scanln(&cfg.Email)
        }
        config.Save(cfg)
    }
    
    // Display configuration
    fmt.Printf("Email: %s\n", cfg.Email)
    fmt.Printf("Device ID: %s\n", cfg.DeviceID)
    fmt.Printf("Resource Mode: %s\n", cfg.ResourceMode)
    fmt.Printf("Background Processing: %v\n", cfg.AllowBackground)
    
    // Show idle status
    idleTime, err := idle.GetIdleTime()
    if err == nil {
        fmt.Printf("Current idle time: %v\n", idleTime)
    }
    
    // Create resource manager
    resourceMgr := resource.NewManager(cfg.ResourceMode)
    cpuLimit, memLimit := resourceMgr.GetLimits()
    fmt.Printf("Resource limits: CPU=%d%%, Memory=%d%%\n", cpuLimit, memLimit)
    
    // Create API client
    apiClient := api.NewClient(cfg.APIBase, cfg.Email, cfg.DeviceID)
    if bypass := os.Getenv("VERCEL_BYPASS_TOKEN"); bypass != "" {
        apiClient.SetBypassToken(bypass)
    }
    
    // Register if needed
    if !cfg.Registered {
        fmt.Print("Registering with server... ")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        err := apiClient.Register(ctx, cfg.Referral, version)
        cancel()
        
        if err != nil {
            fmt.Printf("Failed: %v\n", err)
        } else {
            fmt.Println("Success!")
            cfg.Registered = true
            config.Save(cfg)
        }
    }
    
    // Create job executor
    jobExecutor, err := executor.NewExecutor(resourceMgr)
    if err != nil {
        fmt.Printf("Failed to create job executor: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("========================================")
    
    // Setup shutdown handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Start tickers
    heartbeatTicker := time.NewTicker(30 * time.Second)
    defer heartbeatTicker.Stop()
    
    jobTicker := time.NewTicker(20 * time.Second)
    defer jobTicker.Stop()
    
    cleanupTicker := time.NewTicker(1 * time.Hour)
    defer cleanupTicker.Stop()
    
    statusTicker := time.NewTicker(1 * time.Minute)
    defer statusTicker.Stop()
    
    fmt.Println("Agent running. Press Ctrl+C to stop.")
    
    // Main event loop
    for {
        select {
        case <-ctx.Done():
            fmt.Println("\nShutting down...")
            return
            
        case <-sigChan:
            fmt.Println("\nShutdown signal received")
            cancel()
            
        case <-heartbeatTicker.C:
            timestamp := time.Now().Format("15:04:05")
            beatCtx, beatCancel := context.WithTimeout(ctx, 5*time.Second)
            err := apiClient.Beat(beatCtx)
            beatCancel()
            
            if err != nil {
                fmt.Printf("[%s] Heartbeat failed: %v\n", timestamp, err)
            } else {
                fmt.Printf("[%s] Heartbeat OK\n", timestamp)
            }
            
        case <-jobTicker.C:
            // Check if we should run jobs
            if !resourceMgr.ShouldRunJob() {
                continue
            }
            
            timestamp := time.Now().Format("15:04:05")
            
            // Check for available jobs
            jobCtx, jobCancel := context.WithTimeout(ctx, 5*time.Second)
            job, err := apiClient.GetNextJob(jobCtx)
            jobCancel()
            
            if err != nil {
                fmt.Printf("[%s] Job check failed: %v\n", timestamp, err)
            } else if job != nil {
                fmt.Printf("[%s] Got job %s\n", timestamp, job.ID)
                
                // Execute job
                execCtx, execCancel := context.WithTimeout(ctx, time.Duration(job.MaxSeconds)*time.Second)
                result, err := jobExecutor.ExecuteJob(execCtx, job.ID, job.ArtifactURL, job.SHA256, job.MaxSeconds)
                execCancel()
                
                if err != nil {
                    fmt.Printf("[%s] Job execution error: %v\n", timestamp, err)
                } else {
                    fmt.Printf("[%s] Job %s completed: success=%v, duration=%v\n", 
                        timestamp, job.ID, result.Success, result.EndTime.Sub(result.StartTime))
                }
            }
            
        case <-cleanupTicker.C:
            // Clean up old job directories
            jobExecutor.CleanupWorkDir()
            
        case <-statusTicker.C:
            // Print status update
            timestamp := time.Now().Format("15:04:05")
            idleTime, _ := idle.GetIdleTime()
            cpuLimit, memLimit := resourceMgr.GetLimits()
            activityLevel, _ := idle.GetActivityLevel()
            
            fmt.Printf("[%s] Status: Idle=%v, Activity=%d%%, Limits=CPU:%d%% MEM:%d%%\n", 
                timestamp, idleTime, activityLevel, cpuLimit, memLimit)
        }
    }
}