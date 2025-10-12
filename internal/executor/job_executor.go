package executor

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

// ExecuteJob runs the downloaded job artifact
func ExecuteJob(ctx context.Context, jobType string, artifactData []byte, args json.RawMessage, timeoutSec int) (map[string]interface{}, error) {
    // Create a timeout context
    timeout := time.Duration(timeoutSec) * time.Second
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    result := make(map[string]interface{})
    
    switch jobType {
    case "javascript", "js":
        // For JavaScript, save to temp file and run with Node.js
        tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("job_%d.js", time.Now().Unix()))
        if err := os.WriteFile(tempFile, artifactData, 0644); err != nil {
            return nil, fmt.Errorf("writing temp file: %w", err)
        }
        defer os.Remove(tempFile)
        
        // Execute with Node.js
        cmd := exec.CommandContext(ctx, "node", tempFile)
        output, err := cmd.CombinedOutput()
        if err != nil {
            result["error"] = err.Error()
            result["output"] = string(output)
            return result, err
        }
        
        result["output"] = string(output)
        result["success"] = true
        
    case "sleep":
        // Keep sleep for testing
        time.Sleep(5 * time.Second)
        result["output"] = "Sleep completed"
        result["success"] = true
        
    default:
        return nil, fmt.Errorf("unsupported job type: %s", jobType)
    }
    
    return result, nil
}
