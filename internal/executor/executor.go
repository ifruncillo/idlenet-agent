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
    
    "github.com/bytecodealliance/wasmtime-go/v15"
    "github.com/ifruncillo/idlenet-agent/internal/resource"
)

// JobExecutor handles the execution of computational jobs
type JobExecutor struct {
    resourceMgr *resource.Manager
    workDir     string
    maxTimeout  time.Duration
    engine      *wasmtime.Engine
    wasiConfig  *wasmtime.WasiConfig
}

// NewExecutor creates a new job executor with WASM support
func NewExecutor(resourceMgr *resource.Manager) (*JobExecutor, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    
    workDir := filepath.Join(homeDir, ".idlenet", "work")
    if err := os.MkdirAll(workDir, 0755); err != nil {
        return nil, err
    }
    
    // Create WASM engine with resource limits
    config := wasmtime.NewConfig()
    config.SetConsumeFuel(true)
    config.SetEpochInterruption(true)
    
    engine := wasmtime.NewEngineWithConfig(config)
    
    return &JobExecutor{
        resourceMgr: resourceMgr,
        workDir:     workDir,
        maxTimeout:  30 * time.Minute,
        engine:      engine,
    }, nil
}

// ExecuteWASM runs a WASM module with sandboxing
func (e *JobExecutor) ExecuteWASM(ctx context.Context, wasmPath string, args []string) (*JobResult, error) {
    result := &JobResult{
        StartTime: time.Now(),
    }
    
    // Read WASM module
    wasmBytes, err := os.ReadFile(wasmPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read WASM: %w", err)
    }
    
    // Create store with fuel limit based on resource manager
    store := wasmtime.NewStore(e.engine)
    store.AddFuel(1000000000) // 1 billion units of fuel
    
    // Compile module
    module, err := wasmtime.NewModule(e.engine, wasmBytes)
    if err != nil {
        result.Error = fmt.Sprintf("Failed to compile WASM: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Setup WASI environment (sandboxed)
    wasiConfig := wasmtime.NewWasiConfig()
    wasiConfig.SetArgv(args)
    wasiConfig.SetStdout(os.Stdout) // In production, capture to file
    wasiConfig.SetStderr(os.Stderr)
    
    // Limit filesystem access to job directory only
    jobDir := filepath.Dir(wasmPath)
    wasiConfig.PreopenDir(jobDir, "/")
    
    store.SetWasi(wasiConfig)
    
    // Create linker and instantiate
    linker := wasmtime.NewLinker(e.engine)
    err = linker.DefineWasi()
    if err != nil {
        result.Error = fmt.Sprintf("Failed to define WASI: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    instance, err := linker.Instantiate(store, module)
    if err != nil {
        result.Error = fmt.Sprintf("Failed to instantiate: %v", err)
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Get the _start function (WASI entry point)
    start := instance.GetFunc(store, "_start")
    if start == nil {
        result.Error = "No _start function found"
        result.EndTime = time.Now()
        return result, nil
    }
    
    // Execute with timeout
    done := make(chan error, 1)
    go func() {
        _, err := start.Call(store)
        done <- err
    }()
    
    select {
    case err := <-done:
        if err != nil {
            result.Error = fmt.Sprintf("Execution error: %v", err)
            result.Success = false
        } else {
            result.Success = true
            result.Output = "WASM execution completed successfully"
        }
    case <-ctx.Done():
        result.Error = "Execution timeout"
        result.Success = false
    }
    
    result.EndTime = time.Now()
    result.CPUTime = result.EndTime.Sub(result.StartTime)
    
    return result, nil
}

// JobResult remains the same
type JobResult struct {
    Success   bool
    Output    string
    Error     string
    StartTime time.Time
    EndTime   time.Time
    CPUTime   time.Duration
}

// ExecuteJob orchestrates the full job execution
func (e *JobExecutor) ExecuteJob(ctx context.Context, jobID, artifactURL, expectedSHA256 string, timeoutSeconds int) (*JobResult, error) {
    if !e.resourceMgr.ShouldRunJob() {
        return nil, fmt.Errorf("system too active to run jobs")
    }
    
    jobDir := filepath.Join(e.workDir, jobID)
    if err := os.MkdirAll(jobDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create job directory: %w", err)
    }
    defer os.RemoveAll(jobDir)
    
    artifactPath := filepath.Join(jobDir, "job.wasm")
    if err := e.downloadAndVerify(artifactURL, artifactPath, expectedSHA256); err != nil {
        result := &JobResult{
            StartTime: time.Now(),
            EndTime:   time.Now(),
            Error:     fmt.Sprintf("Failed to download artifact: %v", err),
            Success:   false,
        }
        return result, nil
    }
    
    timeout := time.Duration(timeoutSeconds) * time.Second
    if timeout > e.maxTimeout {
        timeout = e.maxTimeout
    }
    
    jobCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    return e.ExecuteWASM(jobCtx, artifactPath, []string{})
}

// downloadAndVerify remains the same
func (e *JobExecutor) downloadAndVerify(url, destPath, expectedSHA256 string) error {
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("download failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed with status: %d", resp.StatusCode)
    }
    
    tempPath := destPath + ".tmp"
    tempFile, err := os.Create(tempPath)
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    defer os.Remove(tempPath)
    
    hasher := sha256.New()
    writer := io.MultiWriter(tempFile, hasher)
    
    if _, err := io.Copy(writer, resp.Body); err != nil {
        tempFile.Close()
        return fmt.Errorf("failed to save file: %w", err)
    }
    tempFile.Close()
    
    actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
    if actualSHA256 != expectedSHA256 {
        return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
    }
    
    if err := os.Rename(tempPath, destPath); err != nil {
        return fmt.Errorf("failed to move file: %w", err)
    }
    
    return nil
}

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
            
            if now.Sub(info.ModTime()) > 24*time.Hour {
                os.RemoveAll(path)
            }
        }
    }
    
    return nil
}