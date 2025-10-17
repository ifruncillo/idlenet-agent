package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/ifruncillo/idlenet-agent/internal/api"
    "github.com/ifruncillo/idlenet-agent/internal/executor"
)

type Worker struct {
    client *api.Client
}

func New(client *api.Client, email, deviceID string) *Worker {
    return &Worker{client: client}
}

func (w *Worker) Run(ctx context.Context) error {
    ticker := time.NewTicker(20 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            w.ProcessNextJob(ctx)
        }
    }
}

func (w *Worker) ProcessNextJob(ctx context.Context) {
    // Get next job from the queue
    job, err := w.client.GetNextJob(ctx)
    if err != nil {
        fmt.Printf("Error getting job: %v\n", err)
        return
    }
    if job == nil {
        // No jobs available
        return
    }

    fmt.Printf("Got job %s of type %s\n", job.ID, job.Type)
    startTime := time.Now()

    // Download the artifact if provided
    var artifactData []byte
    if job.ArtifactURL != "" {
        artifactData, err = w.client.DownloadArtifact(ctx, job.ArtifactURL, job.SHA256)
        if err != nil {
            fmt.Printf("Failed to download artifact: %v\n", err)
            w.client.ReportJobComplete(ctx, job.ID, "failed",
                map[string]interface{}{"error": err.Error()}, 0)
            return
        }
        fmt.Printf("Downloaded %d bytes, SHA256 verified\n", len(artifactData))
        if len(artifactData) > 0 {
            preview := artifactData
            if len(preview) > 100 {
                preview = preview[:100]
            }
            fmt.Printf("First %d chars of artifact: %s...\n", len(preview), string(preview))
        }
    }

    // Convert args map back to JSON for executor
    argsJSON, _ := json.Marshal(job.Args)

    // Execute the job using the function, not method
    result, err := executor.ExecuteJob(ctx, job.Type, artifactData, argsJSON, job.MaxSeconds)
    if err != nil {
        fmt.Printf("Job execution failed: %v\n", err)
    }

    // Check if result contains output that should be saved as a file
    resultFilePath := ""
    if output, ok := result["output"].(string); ok && output != "" {
        // Create a result file
        tmpFile, createErr := os.CreateTemp("", fmt.Sprintf("job-%s-result-*.txt", job.ID))
        if createErr == nil {
            tmpFile.WriteString(output)
            tmpFile.Close()
            resultFilePath = tmpFile.Name()
            fmt.Printf("Created result file: %s\n", resultFilePath)
        }
    }

    // Upload result file if one was created
    if resultFilePath != "" {
        fmt.Printf("Uploading result file for job %s...\n", job.ID)
        if uploadErr := w.client.UploadJobResult(ctx, job.ID, resultFilePath); uploadErr != nil {
            fmt.Printf("Failed to upload result: %v\n", uploadErr)
        } else {
            fmt.Printf("Result uploaded successfully for job %s\n", job.ID)
        }
        // Clean up temp file
        os.Remove(resultFilePath)
    }

    // Report completion
    execTime := time.Since(startTime).Milliseconds()
    status := "completed"
    if err != nil {
        status = "failed"
    }
    if err := w.client.ReportJobComplete(ctx, job.ID, status, result, execTime); err != nil {
        fmt.Printf("Failed to report job completion: %v\n", err)
    } else {
        fmt.Printf("Job %s completed in %dms\n", job.ID, execTime)
    }
}
