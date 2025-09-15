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
)

// JobExecutor handles the execution of computational jobs
type JobExecutor struct {
    resourceMgr *resource.Manager
    workDir     string
    maxTimeout  time.Duration
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
    
    return &JobExecutor{
        resourceMgr: resourceMgr,
        workDir:     workDir,
        maxTimeout:  30 * time.Minute,
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
    
    // TODO: Execute WASM with wasmtime-go
    // For now, simulate job execution
    select {
    case <-jobCtx.Done():
        result.Error = "Job timed out"
        result.Success = false
    case <-time.After(5 * time.Second): // Simulate 5 second job
        result.Output = fmt.Sprintf("Job completed successfully using %d%% CPU, %d%% memory on %d cores", 
            cpuLimit, memLimit, cores)
        result.Success = true
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