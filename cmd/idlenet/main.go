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
    "github.com/ifruncillo/idlenet-agent/internal/idle"
    "github.com/ifruncillo/idlenet-agent/internal/metrics"
    "github.com/ifruncillo/idlenet-agent/internal/resource"
)

const version = "v1.0.0"

func main() {
    fmt.Printf("IdleNet Agent %s\n", version)
    fmt.Println("========================================")
    
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("Failed to load configuration: %v\n", err)
        os.Exit(1)
    }
    
    if cfg.Email == "" {
        if email := os.Getenv("IDLENET_EMAIL"); email != "" {
            cfg.Email = email
        } else {
            fmt.Print("Enter your email address: ")
            fmt.Scanln(&cfg.Email)
        }
        config.Save(cfg)
    }
    
    fmt.Printf("Email: %s\n", cfg.Email)
    fmt.Printf("Device ID: %s\n", cfg.DeviceID)
    fmt.Printf("Resource Mode: %s\n", cfg.ResourceMode)
    
    // Initialize metrics tracker
    metricsTracker := metrics.NewTracker()
    perfMonitor := metrics.NewPerformanceMonitor()
    
    idleTime, err := idle.GetIdleTime()
    if err == nil {
        fmt.Printf("Current idle time: %v\n", idleTime)
    }
    
    resourceMgr := resource.NewManager(cfg.ResourceMode)
    cpuLimit, memLimit := resourceMgr.GetLimits()
    fmt.Printf("Resource limits: CPU=%d%%, Memory=%d%%\n", cpuLimit, memLimit)
    
    apiClient := api.NewClient(cfg.APIBase, cfg.Email, cfg.DeviceID)
    
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
    
    // jobExecutor temporarily disabled for testing    }
    
    fmt.Println("========================================")
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    heartbeatTicker := time.NewTicker(30 * time.Second)
    defer heartbeatTicker.Stop()
    
    jobTicker := time.NewTicker(20 * time.Second)
    defer jobTicker.Stop()
    
    statusTicker := time.NewTicker(1 * time.Minute)
    defer statusTicker.Stop()
    
    metricsTicker := time.NewTicker(5 * time.Minute)
    defer metricsTicker.Stop()
    
    fmt.Println("Agent running. Press Ctrl+C to stop.")
    
    for {
        select {
        case <-ctx.Done():
            fmt.Println("\nShutting down...")
            completed, failed, cpuTime, earnings := metricsTracker.GetStats()
            fmt.Printf("Session stats: %d completed, %d failed, CPU time: %v, Earnings: $%.4f\n", 
                completed, failed, cpuTime, earnings)
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
            if !resourceMgr.ShouldRunJob() {
                continue
            }
            
            timestamp := time.Now().Format("15:04:05")
            
            jobCtx, jobCancel := context.WithTimeout(ctx, 5*time.Second)
            job, err := apiClient.GetNextJob(jobCtx)
            jobCancel()
            
            if err != nil {
                fmt.Printf("[%s] Job check failed: %v\n", timestamp, err)
            } else if job != nil {
                fmt.Printf("[%s] Got job %s\n", timestamp, job.ID)
                metricsTracker.RecordJobStart(job.ID)
                
                // Execute job
                jobMetrics := &metrics.JobMetrics{
                    JobID:     job.ID,
                    DeviceID:  cfg.DeviceID,
                    StartTime: time.Now(),
                }
                
                // Simulate job execution (replace with actual execution)
                time.Sleep(2 * time.Second)
                jobMetrics.EndTime = time.Now()
                jobMetrics.Success = true
                jobMetrics.CPUSeconds = 2.0
                jobMetrics.MemoryMB = 256
                
                metricsTracker.RecordJobComplete(jobMetrics)
                
                fmt.Printf("[%s] Job %s completed, earned: $%.4f\n", 
                    timestamp, job.ID, jobMetrics.Earnings)
            }
            
        case <-statusTicker.C:
            timestamp := time.Now().Format("15:04:05")
            idleTime, _ := idle.GetIdleTime()
            cpuLimit, memLimit := resourceMgr.GetLimits()
            
            currentMetrics := metricsTracker.GetCurrentMetrics()
            fmt.Printf("[%s] Status: Idle=%v, Limits=CPU:%d%% MEM:%d%%, Jobs=%d, Earnings=$%.4f\n", 
                timestamp, idleTime, cpuLimit, memLimit, 
                currentMetrics.TotalJobs, currentMetrics.Earnings)
                
        case <-metricsTicker.C:
            // Sample performance and check system health
            sample := perfMonitor.Sample()
            if !perfMonitor.IsSystemHealthy() {
                fmt.Println("Warning: System performance impact detected")
            }
            _ = sample // Use sample data as needed
        }
    }
}