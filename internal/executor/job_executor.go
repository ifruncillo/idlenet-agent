package executor

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
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
    case "javascript", "js", "compute", "user-upload":
        // For JavaScript, save to temp file and run with Node.js
        // Use os.CreateTemp to avoid name collisions
        tmpFile, err := os.CreateTemp("", "job-*.js")
        if err != nil {
            return nil, fmt.Errorf("creating temp file: %w", err)
        }
        tempFile := tmpFile.Name()

        if _, err := tmpFile.Write(artifactData); err != nil {
            tmpFile.Close()
            os.Remove(tempFile)
            return nil, fmt.Errorf("writing temp file: %w", err)
        }
        tmpFile.Close()
        defer os.Remove(tempFile)

        // Execute with Node.js with memory limit (512MB default)
        // This provides basic resource protection
        cmd := exec.CommandContext(ctx, "node", "--max-old-space-size=512", tempFile)
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

