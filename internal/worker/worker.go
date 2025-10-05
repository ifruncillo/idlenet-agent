package worker

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/ifruncillo/idlenet-agent/internal/api"
    "github.com/ifruncillo/idlenet-agent/internal/executor"
)

type Worker struct {
    client *api.Client
}

func New(client *api.Client) *Worker {
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
            w.processNextJob(ctx)
        }
    }
}

func (w *Worker) processNextJob(ctx context.Context) {
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
    
    fmt.Printf("Got job %s of type %s\n", job.JobID, job.Type)
    startTime := time.Now()
    
    // Download the artifact
    artifactData, err := w.client.DownloadArtifact(ctx, job.ArtifactURL, job.ArtifactSHA256)
    if err != nil {
        fmt.Printf("Failed to download artifact: %v\n", err)
        w.client.ReportJobComplete(ctx, job.JobID, "failed", 
            map[string]interface{}{"error": err.Error()}, 0)
        return
    }
    
    fmt.Printf("Downloaded %d bytes, SHA256 verified\n", len(artifactData))
    
    // Execute the job
    result, err := executor.ExecuteJob(ctx, job.Type, artifactData, job.Args, job.TimeoutSec)
    if err != nil {
        fmt.Printf("Job execution failed: %v\n", err)
    }
    
    // Report completion
    execTime := time.Since(startTime).Milliseconds()
    status := "completed"
    if err != nil {
        status = "failed"
    }
    
    if err := w.client.ReportJobComplete(ctx, job.JobID, status, result, execTime); err != nil {
        fmt.Printf("Failed to report job completion: %v\n", err)
    } else {
        fmt.Printf("Job %s completed in %dms\n", job.JobID, execTime)
    }
}
