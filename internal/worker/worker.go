package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/ifruncillo/idlenet-agent/internal/api"
    "github.com/ifruncillo/idlenet-agent/internal/executor"
    "github.com/ifruncillo/idlenet-agent/internal/metrics"
)

type Worker struct {
    client  *api.Client
    tracker *metrics.Tracker
}

func New(client *api.Client, email, deviceID string) *Worker {
    return &Worker{
        client: client,
        tracker: metrics.NewTracker(),
    }
}

func (w *Worker) SetTracker(tracker *metrics.Tracker) {
    w.tracker = tracker
}

func (w *Worker) ProcessNextJob(ctx context.Context) {
    job, err := w.client.GetNextJob(ctx)
    if err != nil {
        fmt.Printf("Error getting job: %v\n", err)
        return
    }
    if job == nil {
        return
    }

    fmt.Printf("Got job %s of type %s\n", job.ID, job.Type)
    startTime := time.Now()

    // Record job start
    if w.tracker != nil {
        w.tracker.RecordJobStart(job.ID)
    }

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
    }

    argsJSON, _ := json.Marshal(job.Args)
    result, err := executor.ExecuteJob(ctx, job.Type, artifactData, argsJSON, job.MaxSeconds)
    
    success := err == nil
    if err != nil {
        fmt.Printf("Job execution failed: %v\n", err)
    }

    // Only create temp file for large outputs (>100KB) to reduce disk I/O
    resultFilePath := ""
    if output, ok := result["output"].(string); ok && output != "" && len(output) > 100*1024 {
        tmpFile, createErr := os.CreateTemp("", fmt.Sprintf("job-%s-result-*.txt", job.ID))
        if createErr != nil {
            fmt.Printf("Warning: failed to create result file: %v\n", createErr)
        } else {
            if _, writeErr := tmpFile.WriteString(output); writeErr != nil {
                tmpFile.Close()
                os.Remove(tmpFile.Name())
                fmt.Printf("Warning: failed to write result file: %v\n", writeErr)
            } else {
                tmpFile.Close()
                resultFilePath = tmpFile.Name()
                fmt.Printf("Created result file: %s (%d bytes)\n", resultFilePath, len(output))
            }
        }
    }

    if resultFilePath != "" {
        fmt.Printf("Uploading result file for job %s...\n", job.ID)
        if uploadErr := w.client.UploadJobResult(ctx, job.ID, resultFilePath); uploadErr != nil {
            fmt.Printf("Failed to upload result: %v\n", uploadErr)
        } else {
            fmt.Printf("Result uploaded successfully for job %s\n", job.ID)
        }
        os.Remove(resultFilePath)
    }

    execTime := time.Since(startTime).Milliseconds()
    status := "completed"
    if err != nil {
        status = "failed"
    }

    // Record job completion with metrics
    if w.tracker != nil {
        jobMetrics := &metrics.JobMetrics{
            JobID:     job.ID,
            DeviceID:  w.client.DeviceID,
            StartTime: startTime,
            EndTime:   time.Now(),
            CPUSeconds: time.Since(startTime).Seconds(),
            Success:   success,
        }
        if err != nil {
            jobMetrics.ErrorMessage = err.Error()
        }
        w.tracker.RecordJobComplete(jobMetrics)
    }

    if err := w.client.ReportJobComplete(ctx, job.ID, status, result, execTime); err != nil {
        fmt.Printf("Failed to report job completion: %v\n", err)
    } else {
        fmt.Printf("Job %s completed in %dms\n", job.ID, execTime)
    }
}
