# IdleNet Agent - Inefficiency Scan Report

**Generated:** 2025-11-02
**Codebase:** IdleNet Agent v1.0.0
**Language:** Go 1.22

## Executive Summary

This report identifies performance inefficiencies, code redundancies, and optimization opportunities across the IdleNet Agent codebase. The scan covers ~2,200 lines of Go code across 21 files, focusing on:

- API communication patterns
- Resource management
- File I/O operations
- Concurrency issues
- Memory allocations
- Code duplication

## Critical Issues (High Priority)

### 1. Missing Context in HTTP Downloads
**Location:** `internal/api/client.go:200`
**Severity:** HIGH

```go
resp, err := http.Get(artifactURL)  // Line 200
```

**Issue:** Regular URL artifact downloads use `http.Get()` without context, ignoring the timeout and cancellation context passed to the function. This can cause downloads to hang indefinitely.

**Impact:**
- Jobs can't be cancelled mid-download
- Ignores the overall job timeout context
- Potential resource leak if download hangs

**Recommendation:**
```go
req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
if err != nil {
    return nil, err
}
resp, err := c.httpClient.Do(req)
```

---

### 2. Double JSON Marshaling in API Calls
**Location:** `internal/api/client.go:232-233`
**Severity:** HIGH

```go
jsonBody, _ := json.Marshal(body)  // Line 232
resp, err := c.doRequest(ctx, "POST", "/api/jobs/complete", bytes.NewReader(jsonBody))  // Line 233
```

**Issue:** `ReportJobComplete` marshals the payload to JSON, then passes it to `doRequest()` which expects a map and marshals it again. The JSON bytes are wrapped in an `io.Reader` but `doRequest()` expects `interface{}` and will marshal it incorrectly.

**Impact:**
- Incorrect data sent to API
- Wasted CPU on redundant marshaling
- Potential API errors due to malformed requests

**Recommendation:**
```go
resp, err := c.doRequest(ctx, "POST", "/api/jobs/complete", body)  // Pass map directly
```

---

### 3. No Resource Limits on Node.js Execution
**Location:** `internal/executor/job_executor.go:32`
**Severity:** HIGH

```go
cmd := exec.CommandContext(ctx, "node", tempFile)
output, err := cmd.CombinedOutput()
```

**Issue:** Node.js processes are spawned without CPU or memory limits. A malicious or buggy job could consume 100% CPU or all available memory, despite the resource manager's limits.

**Impact:**
- Resource limits configured by user are ignored during execution
- System can become unresponsive
- Defeats the purpose of the resource manager

**Recommendation:**
Use cgroups (Linux) or job objects (Windows) to enforce limits, or at minimum use node's `--max-old-space-size` flag:
```go
cpuLimit, memLimit := resourceMgr.GetLimits()
cmd := exec.CommandContext(ctx, "node",
    fmt.Sprintf("--max-old-space-size=%d", memLimit),
    tempFile)
```

---

### 4. Inefficient File I/O for Every Job Result
**Location:** `internal/worker/worker.go:78-84`
**Severity:** MEDIUM

```go
tmpFile, createErr := os.CreateTemp("", fmt.Sprintf("job-%s-result-*.txt", job.ID))
if createErr == nil {
    tmpFile.WriteString(output)  // No error check
    tmpFile.Close()              // No error check
    resultFilePath = tmpFile.Name()
}
```

**Issue:** Creates a temporary file for EVERY job result, even for small outputs that could be sent directly in the API call. Also missing error handling for write operations.

**Impact:**
- Unnecessary disk I/O for every job
- Potential data loss if write fails silently
- Temp directory pollution

**Recommendation:**
Only write to file for large results:
```go
if len(output) > 1024*100 { // Only for >100KB outputs
    tmpFile, err := os.CreateTemp("", fmt.Sprintf("job-%s-*.txt", job.ID))
    if err != nil {
        return fmt.Errorf("create temp: %w", err)
    }
    defer os.Remove(tmpFile.Name())
    if _, err := tmpFile.WriteString(output); err != nil {
        tmpFile.Close()
        return fmt.Errorf("write result: %w", err)
    }
    tmpFile.Close()
    resultFilePath = tmpFile.Name()
}
```

---

## Performance Issues (Medium Priority)

### 5. File Opened/Closed for Every Job Metric
**Location:** `internal/metrics/tracker.go:117-135`
**Severity:** MEDIUM

```go
func (t *Tracker) saveJobMetrics(job *JobMetrics) {
    file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return
    }
    defer file.Close()

    data, _ := json.Marshal(job)
    file.Write(data)
    file.WriteString("\n")
}
```

**Issue:** Opens and closes the metrics file for every single job completion. For high job throughput, this creates excessive syscalls.

**Impact:**
- Significant I/O overhead for frequent jobs
- File system contention
- Slower job completion reporting

**Recommendation:**
Buffer metrics writes and flush periodically:
```go
// Keep file handle open, buffer writes, flush every 10 jobs or 30 seconds
type metricsBuffer struct {
    jobs []*JobMetrics
    mu   sync.Mutex
}

func (t *Tracker) saveJobMetrics(job *JobMetrics) {
    t.buffer.mu.Lock()
    t.buffer.jobs = append(t.buffer.jobs, job)
    shouldFlush := len(t.buffer.jobs) >= 10
    t.buffer.mu.Unlock()

    if shouldFlush {
        t.flushMetrics()
    }
}
```

---

### 6. Duplicate URL Building Logic
**Location:** `internal/api/client.go:97-108, 266-274`
**Severity:** MEDIUM

The Vercel bypass token URL building logic is duplicated in two places:
- `doRequest()` lines 97-108
- `UploadJobResult()` lines 266-274

**Impact:**
- Code duplication (~12 lines)
- Maintenance burden (must update both places)
- Potential for bugs if one is updated and the other isn't

**Recommendation:**
Extract to helper method:
```go
func (c *Client) addBypassParams(fullURL string) (string, error) {
    if c.bypass == "" {
        return fullURL, nil
    }

    parsed, err := url.Parse(fullURL)
    if err != nil {
        return "", err
    }

    query := parsed.Query()
    query.Set("x-vercel-set-bypass-cookie", "true")
    query.Set("x-vercel-protection-bypass", c.bypass)
    parsed.RawQuery = query.Encode()
    return parsed.String(), nil
}
```

---

### 7. Unnecessary String Preview Generation
**Location:** `internal/worker/worker.go:59-65`
**Severity:** LOW

```go
if len(artifactData) > 0 {
    preview := artifactData
    if len(preview) > 100 {
        preview = preview[:100]
    }
    fmt.Printf("First %d chars of artifact: %s...\n", len(preview), string(preview))
}
```

**Issue:** Converts binary data to string for preview printing on every job. This is inefficient for binary artifacts and provides minimal debugging value in production.

**Impact:**
- Unnecessary memory allocation for string conversion
- Prints garbage for binary artifacts
- Clutters log output

**Recommendation:**
Remove or gate behind a debug flag:
```go
if debug && len(artifactData) > 0 {
    preview := artifactData
    if len(preview) > 100 {
        preview = preview[:100]
    }
    fmt.Printf("Artifact preview: %s...\n", string(preview))
}
```

---

### 8. Redundant Struct Copy in GetCurrentMetrics
**Location:** `internal/metrics/tracker.go:95-96`
**Severity:** LOW

```go
func (t *Tracker) GetCurrentMetrics() *SystemMetrics {
    t.mu.RLock()
    defer t.mu.RUnlock()

    metrics := *t.currentMetrics  // Full struct copy
    metrics.Timestamp = time.Now()
    metrics.SessionHours = time.Since(t.sessionStart).Hours()
    metrics.Earnings = t.totalEarnings

    return &metrics
}
```

**Issue:** Creates a full copy of SystemMetrics struct just to update a few fields. While not expensive for this small struct, it's unnecessary.

**Impact:**
- Minor memory allocation overhead
- Called frequently (every status tick)

**Recommendation:**
```go
return &SystemMetrics{
    Timestamp:    time.Now(),
    CPUPercent:   t.currentMetrics.CPUPercent,
    MemoryMB:     t.currentMetrics.MemoryMB,
    JobsRunning:  t.currentMetrics.JobsRunning,
    TotalJobs:    t.jobsCompleted + t.jobsFailed,
    SessionHours: time.Since(t.sessionStart).Hours(),
    Earnings:     t.totalEarnings,
}
```

---

## Resource Management Issues

### 9. Unused Performance Monitor
**Location:** `cmd/idlenet/main.go:47-48, 155-161`
**Severity:** MEDIUM

```go
perfMonitor := metrics.NewPerformanceMonitor()  // Line 48

// Later, in event loop:
case <-metricsTicker.C:
    sample := perfMonitor.Sample()
    if !perfMonitor.IsSystemHealthy() {
        fmt.Println("Warning: System performance impact detected")
    }
    _ = sample // Use sample data as needed  // Line 161
```

**Issue:**
- PerformanceMonitor is created but barely used
- Sample data is collected then explicitly ignored
- IsSystemHealthy() always returns true because CPUPercent is always 0 (performance.go:35)
- Health check has no action taken except a print statement

**Impact:**
- Wasted memory for 60 samples ring buffer
- Wasted CPU on runtime.ReadMemStats() every 5 minutes
- False sense of monitoring

**Recommendation:**
Either:
1. Remove PerformanceMonitor entirely if not needed
2. Actually implement CPU monitoring and take action on health issues:
```go
if !perfMonitor.IsSystemHealthy() {
    fmt.Println("System health degraded - pausing new jobs")
    resourceMgr.SetPaused(true)
    time.AfterFunc(5*time.Minute, func() { resourceMgr.SetPaused(false) })
}
```

---

### 10. Inefficient Limit Caching Check
**Location:** `cmd/idlenet/main.go:148`
**Severity:** LOW

```go
case <-statusTicker.C:
    timestamp := time.Now().Format("15:04:05")
    idleTime, _ := idle.GetIdleTime()
    cpuLimit, memLimit := resourceMgr.GetLimits()  // Called here
```

**Issue:** Calls `GetLimits()` every status tick (60 seconds) even though it has a 5-second cache. Not a major issue, but the method is called just for display purposes.

**Impact:**
- Minor: Unnecessary mutex acquisition every minute

**Recommendation:**
Display the last known limits without calling the method, or accept this minor overhead as acceptable.

---

### 11. History File Written on Every Data Point
**Location:** `internal/dashboard/history.go:42-66`
**Severity:** MEDIUM

```go
func (h *History) Add(earnings float64, jobs int) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.Points = append(h.Points, DataPoint{...})

    // Keep only last 30 days
    cutoff := time.Now().AddDate(0, 0, -30)
    var filtered []DataPoint
    for _, p := range h.Points {
        if p.Time.After(cutoff) {
            filtered = append(filtered, p)
        }
    }
    h.Points = filtered
    h.save()  // Writes to disk on every add!
}
```

**Issue:**
- Writes entire history file to disk every minute
- Filters entire history array every minute (O(n) operation)
- As history grows, this becomes increasingly expensive

**Impact:**
- Unnecessary disk I/O every minute for 30 days (43,200 data points)
- CPU overhead for filtering
- SSD wear

**Recommendation:**
```go
func (h *History) Add(earnings float64, jobs int) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.Points = append(h.Points, DataPoint{...})
    h.dirty = true
}

// Separate goroutine flushes every 10 minutes and cleans daily
func (h *History) startPersistence() {
    ticker := time.NewTicker(10 * time.Minute)
    for range ticker.C {
        h.mu.Lock()
        if h.dirty {
            h.save()
            h.dirty = false
        }
        h.mu.Unlock()
    }
}
```

---

## Code Quality Issues

### 12. Primitive Laptop Detection
**Location:** `internal/resource/manager.go:142`
**Severity:** LOW

```go
func isLaptop() bool {
    return runtime.NumCPU() <= 8
}
```

**Issue:** Detects laptops by assuming ≤8 cores = laptop. Modern laptops can have 16+ cores, and desktops can have 4-6 cores.

**Impact:**
- Incorrect resource limits on modern hardware
- Desktops might get laptop limits
- High-end laptops get desktop limits

**Recommendation:**
Use proper platform detection:
- Linux: Check `/sys/class/power_supply/BAT*/present`
- Windows: Check Win32 `SystemPowerCapabilities`
- macOS: Check `IOPlatformExpertDevice` for battery

Or remove the distinction entirely if not critical.

---

### 13. Browser Auto-Launch Cannot Be Disabled
**Location:** `internal/dashboard/server.go:62-65`
**Severity:** LOW

```go
go func() {
    time.Sleep(2 * time.Second)
    s.openBrowser(fmt.Sprintf("http://localhost:%d", s.port))
}()
```

**Issue:** Dashboard always auto-opens browser after 2 seconds. No way to disable for:
- Headless servers
- CI/CD environments
- Users who don't want it

**Impact:**
- Annoyance for users
- Errors in headless environments
- Unprofessional behavior

**Recommendation:**
```go
if cfg.AutoOpenDashboard {  // Add config option
    go func() {
        time.Sleep(2 * time.Second)
        s.openBrowser(fmt.Sprintf("http://localhost:%d", s.port))
    }()
}
```

---

### 14. Missing Error Handling in Browser Launch
**Location:** `internal/dashboard/server.go:111`
**Severity:** LOW

```go
cmd.Start()  // No error check
```

**Issue:** Browser launch errors are silently ignored.

**Impact:**
- Silent failures on systems without browser
- No feedback to user

**Recommendation:**
```go
if err := cmd.Start(); err != nil {
    fmt.Printf("Could not open browser: %v\n", err)
}
```

---

### 15. Incomplete Idle Detection on Linux/macOS
**Location:** `internal/idle/idle_others.go:11-14`
**Severity:** MEDIUM

```go
// TODO: Implement actual idle detection for macOS and Linux
func GetIdleTime() (time.Duration, error) {
    return 30 * time.Second, nil
}

func GetActivityLevel() (int, error) {
    return 50, nil  // Always returns 50%
}
```

**Issue:** Non-Windows platforms return hardcoded values, making the entire resource management system ineffective.

**Impact:**
- Linux/macOS users get incorrect resource limiting
- Jobs may run when system is actively in use
- Idle-only mode doesn't work

**Recommendation:**
Implement proper idle detection:
- **Linux:** Read `/proc/interrupts` or use X11 `XScreenSaverQueryInfo`
- **macOS:** Use IOKit `CGEventSourceSecondsSinceLastEventType`

---

### 16. Unsafe Config Save with Fallback
**Location:** `internal/config/config.go:118-128`
**Severity:** MEDIUM

```go
tempPath := configPath + ".tmp"
if err := os.WriteFile(tempPath, data, 0644); err != nil {
    return fmt.Errorf("failed to write config: %w", err)
}

if err := os.Rename(tempPath, configPath); err != nil {
    os.Remove(tempPath)  // Delete temp file
    if err := os.WriteFile(configPath, data, 0644); err != nil {  // Direct write!
        return fmt.Errorf("failed to save config: %w", err)
    }
}
```

**Issue:** Falls back to direct write if atomic rename fails. This can corrupt the config file if the process crashes during write.

**Impact:**
- Config corruption on Windows (where rename can fail if file is open)
- Lost user settings

**Recommendation:**
```go
if err := os.Rename(tempPath, configPath); err != nil {
    // On Windows, delete target and try again
    os.Remove(configPath)
    if err := os.Rename(tempPath, configPath); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }
}
```

---

### 17. Temp File Name Collision Risk
**Location:** `internal/executor/job_executor.go:25`
**Severity:** LOW

```go
tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("job_%d.js", time.Now().Unix()))
```

**Issue:** Uses Unix timestamp (1-second granularity) for temp file naming. If two jobs start in the same second, they'll overwrite each other's file.

**Impact:**
- Race condition if jobs execute in parallel
- Job failure with cryptic errors

**Recommendation:**
Use `os.CreateTemp()` for guaranteed unique names:
```go
tmpFile, err := os.CreateTemp("", "job-*.js")
if err != nil {
    return nil, fmt.Errorf("create temp: %w", err)
}
tmpFile.Close()
tempFile := tmpFile.Name()
if err := os.WriteFile(tempFile, artifactData, 0644); err != nil {
    return nil, fmt.Errorf("write artifact: %w", err)
}
defer os.Remove(tempFile)
```

---

### 18. Missing Error Checks in History
**Location:** `internal/dashboard/history.go:36-44`
**Severity:** LOW

```go
func (h *History) load() {
    data, err := os.ReadFile(h.file)
    if err == nil {
        json.Unmarshal(data, &h.Points)  // No error check
    }
}

func (h *History) save() {
    data, _ := json.MarshalIndent(h.Points, "", "  ")  // Ignores error
    os.WriteFile(h.file, data, 0644)                    // Ignores error
}
```

**Issue:** JSON unmarshal errors and file write errors are silently ignored.

**Impact:**
- Silent data loss
- Corrupted history file goes undetected

**Recommendation:**
```go
func (h *History) load() {
    data, err := os.ReadFile(h.file)
    if err != nil {
        return
    }
    if err := json.Unmarshal(data, &h.Points); err != nil {
        log.Printf("Warning: corrupted history file: %v", err)
    }
}
```

---

## Memory Allocation Issues

### 19. Inefficient Array Filtering
**Location:** `internal/dashboard/history.go:57-65`
**Severity:** LOW

```go
cutoff := time.Now().AddDate(0, 0, -30)
var filtered []DataPoint
for _, p := range h.Points {
    if p.Time.After(cutoff) {
        filtered = append(filtered, p)
    }
}
h.Points = filtered
```

**Issue:** Creates new slice without pre-allocating capacity. If history has 10,000 points and 9,000 are retained, the slice will reallocate multiple times.

**Impact:**
- Multiple memory allocations
- Garbage collection pressure

**Recommendation:**
```go
filtered := make([]DataPoint, 0, len(h.Points))  // Pre-allocate
for _, p := range h.Points {
    if p.Time.After(cutoff) {
        filtered = append(filtered, p)
    }
}
```

Or use in-place filtering:
```go
n := 0
for i, p := range h.Points {
    if p.Time.After(cutoff) {
        h.Points[n] = h.Points[i]
        n++
    }
}
h.Points = h.Points[:n]
```

---

### 20. Slice Growth in Performance Monitor
**Location:** `internal/metrics/performance.go:43-47`
**Severity:** LOW

```go
func (pm *PerformanceMonitor) addSample(s PerformanceSample) {
    pm.samples = append(pm.samples, s)
    if len(pm.samples) > pm.maxSamples {
        pm.samples = pm.samples[1:]  // Shift entire slice
    }
}
```

**Issue:** Uses slice shifting to maintain ring buffer. This copies all 60 samples every time once full.

**Impact:**
- O(n) operation on every sample
- Unnecessary memory copying

**Recommendation:**
Use proper ring buffer:
```go
type PerformanceMonitor struct {
    samples []PerformanceSample
    head    int
    count   int
}

func (pm *PerformanceMonitor) addSample(s PerformanceSample) {
    if pm.count < len(pm.samples) {
        pm.samples[pm.count] = s
        pm.count++
    } else {
        pm.samples[pm.head] = s
        pm.head = (pm.head + 1) % len(pm.samples)
    }
}
```

---

## Summary Statistics

| Category | Count |
|----------|-------|
| **Critical Issues** | 4 |
| **High Priority** | 7 |
| **Medium Priority** | 6 |
| **Low Priority** | 3 |
| **Total Issues** | 20 |

### Estimated Impact

| Issue | LOC to Fix | Performance Gain | Risk Reduction |
|-------|-----------|------------------|----------------|
| #1 Context in downloads | 5 | Medium | High |
| #2 Double marshaling | 2 | High | High |
| #3 Resource limits | 15 | High | High |
| #4 Temp file every job | 20 | Medium | Medium |
| #5 Metrics file I/O | 30 | High | Low |
| #6 URL duplication | 15 | Low | Low |
| #11 History writes | 25 | Medium | Medium |
| #15 Idle detection | 50 | N/A | High |

### Recommended Priorities

1. **Week 1:** Fix issues #1, #2, #3 (critical security/correctness)
2. **Week 2:** Fix issues #4, #5, #11 (performance bottlenecks)
3. **Week 3:** Fix issue #15 (Linux/macOS functionality)
4. **Week 4:** Address remaining issues

---

## Additional Recommendations

### Monitoring & Observability
1. Add structured logging (zerolog, zap)
2. Export metrics in Prometheus format
3. Add distributed tracing for job lifecycle

### Testing
1. Add benchmarks for hot paths (job execution, metrics recording)
2. Add load tests for concurrent job execution
3. Add integration tests for resource limiting

### Architecture
1. Consider using a job queue (channel-based) instead of polling every 20s
2. Implement graceful shutdown with job completion wait
3. Add circuit breaker for API failures

### Security
1. Validate artifact SHA256 before execution (currently only downloads)
2. Add timeout enforcement at OS level (cgroups/job objects)
3. Sanitize job output before file write

---

## Conclusion

The codebase is well-structured with clear separation of concerns, but has several efficiency issues primarily around:
- **I/O operations** (files opened too frequently)
- **Resource management** (limits not enforced on execution)
- **API communication** (missing context, double marshaling)
- **Platform support** (incomplete idle detection)

Addressing the critical and high-priority issues would significantly improve performance, reliability, and cross-platform support.
