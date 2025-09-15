package executor

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "time"
    
    "github.com/ifruncillo/idlenet-agent/internal/resource"
    "github.com/ifruncillo/idlenet-agent/internal/wasm"
)

// JobExecutor handles the execution of computational jobs
type JobExecutor struct {
    resourceMgr *resource.Manager
    workDir     string
    maxTimeout  time.Duration
    sandbox     *wasm.Sandbox
}

// NewExecutor creates a new job executor
func NewExecutor(resourceMgr *resource.Manager) (*JobExecutor, error) {
    // Create work directory for job execution
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    
    workDir := filepath.Join(homeDir, ".idlenet", "work")
    if err := os.MkdirAll(workDir, 0755); err != nil {
        return nil, err
    }
    
    // Create WASM sandbox with secure configuration
    sandboxConfig := wasm.DefaultSandboxConfig()
    // Adjust limits based on resource manager settings
    cpuLimit, _ := resourceMgr.GetLimits()
    if cpuLimit < 50 {
        // Reduce WASM limits for low-resource mode
        sandboxConfig.MaxExecutionTime = 15 * time.Second
        sandboxConfig.CPUTimeLimit = 5 * time.Second
        sandboxConfig.MaxMemoryPages = 32 // 2MB
    }
    
    sandbox, err := wasm.NewSandbox(sandboxConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create WASM sandbox: %w", err)
    }
    
    return &JobExecutor{
        resourceMgr: resourceMgr,
        workDir:     workDir,
        maxTimeout:  30 * time.Minute,
        sandbox:     sandbox,
    }, nil
}

// JobResult contains the outcome of a job execution
type JobResult struct {
    Success   bool
    Output    string
    Error     string
    StartTime time.Time
    EndTime   time.Time
    CPUTime   time.Duration
}

// ExecuteJob downloads and runs a computational job
func (e *JobExecutor) ExecuteJob(ctx context.Context, jobID, artifactURL, expectedSHA256 string, timeoutSeconds int) (*JobResult, error) {
    result := &JobResult{
        StartTime: time.Now(),
    }
    
    // Check if we should run jobs based on current system activity
    if !e.resourceMgr.ShouldRunJob() {
        return nil, fmt.Errorf("system too active to run jobs")
    }
    
    // Create job-specific directory
    jobDir := filepath.Join(e.workDir, jobID)
    if err := os.MkdirAll(jobDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create job directory: %w", err)
    }
    defer os.RemoveAll(jobDir) // Clean up after job completes
    
    // Download artifact
    artifactPath := filepath.Join(jobDir, "job.wasm")
    if err := e.downloadAndVerify(artifactURL, artifactPath, expectedSHA256); err != nil {
        result.Error = fmt.Sprintf("Failed to download artifact: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Read and verify WASM file
    wasmBytes, err := os.ReadFile(artifactPath)
    if err != nil {
        result.Error = fmt.Sprintf("Failed to read WASM file: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Verify WASM format and security
    if err := e.sandbox.VerifyWASM(wasmBytes); err != nil {
        result.Error = fmt.Sprintf("WASM verification failed: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Set timeout for job execution
    timeout := time.Duration(timeoutSeconds) * time.Second
    if timeout > e.maxTimeout {
        timeout = e.maxTimeout
    }
    
    jobCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // Get resource limits
    cpuLimit, memLimit := e.resourceMgr.GetLimits()
    cores := e.resourceMgr.GetCoreCount()
    
    // Execute WASM with security sandbox
    wasmResult, err := e.sandbox.Execute(jobCtx, wasmBytes, "main", []interface{}{})
    if err != nil {
        result.Error = fmt.Sprintf("WASM execution setup failed: %v", err)
        result.Success = false
    } else {
        result.Success = wasmResult.Success
        result.CPUTime = wasmResult.CPUTime
        
        if wasmResult.Success {
            result.Output = fmt.Sprintf("WASM job completed successfully. %s. Resource usage: %d%% CPU, %d%% memory on %d cores, Fuel used: %d", 
                wasmResult.Output, cpuLimit, memLimit, cores, wasmResult.FuelUsed)
        } else {
            result.Error = wasmResult.Error
        }
    }
    
    result.EndTime = time.Now()
    result.CPUTime = result.EndTime.Sub(result.StartTime)
    
    return result, nil
}

// downloadAndVerify downloads a file and verifies its SHA256 checksum
func (e *JobExecutor) downloadAndVerify(url, destPath, expectedSHA256 string) error {
    // Download file
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("download failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed with status: %d", resp.StatusCode)
    }
    
    // Create temporary file
    tempPath := destPath + ".tmp"
    tempFile, err := os.Create(tempPath)
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    defer os.Remove(tempPath)
    
    // Calculate SHA256 while downloading
    hasher := sha256.New()
    writer := io.MultiWriter(tempFile, hasher)
    
    if _, err := io.Copy(writer, resp.Body); err != nil {
        tempFile.Close()
        return fmt.Errorf("failed to save file: %w", err)
    }
    tempFile.Close()
    
    // Verify checksum
    actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
    if actualSHA256 != expectedSHA256 {
        return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
    }
    
    // Move to final location
    if err := os.Rename(tempPath, destPath); err != nil {
        return fmt.Errorf("failed to move file: %w", err)
    }
    
    return nil
}

// CleanupWorkDir removes old job directories
func (e *JobExecutor) CleanupWorkDir() error {
    entries, err := os.ReadDir(e.workDir)
    if err != nil {
        return err
    }
    
    now := time.Now()
    for _, entry := range entries {
        if entry.IsDir() {
            path := filepath.Join(e.workDir, entry.Name())
            info, err := entry.Info()
            if err != nil {
                continue
            }
            
            // Remove directories older than 24 hours
            if now.Sub(info.ModTime()) > 24*time.Hour {
                os.RemoveAll(path)
            }
        }
    }
    
    return nil
}

// Close cleans up the executor resources
func (e *JobExecutor) Close() error {
    if e.sandbox != nil {
        return e.sandbox.Close()
    }
    return nil
}