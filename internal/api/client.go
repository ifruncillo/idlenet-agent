package api

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// Client handles all communication with the IdleNet API
type Client struct {
    baseURL    string
    httpClient *http.Client
    email      string
    DeviceID   string
    bypass     string  // Optional Vercel bypass token for protected deployments
}

// NewClient creates a new API client with the given configuration
func NewClient(baseURL, email, DeviceID string) *Client {
    return &Client{
        baseURL:  baseURL,
        email:    email,
        DeviceID: DeviceID,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,  // Don't wait forever for responses
        },
    }
}

// SetBypassToken sets the Vercel bypass token if the deployment is protected
func (c *Client) SetBypassToken(token string) {
    c.bypass = token
}

// addBypassParams adds Vercel bypass parameters to a URL if bypass token is set
func (c *Client) addBypassParams(fullURL string) (string, error) {
    if c.bypass == "" {
        return fullURL, nil
    }

    parsed, err := url.Parse(fullURL)
    if err != nil {
        return "", fmt.Errorf("invalid URL: %w", err)
    }

    query := parsed.Query()
    query.Set("x-vercel-set-bypass-cookie", "true")
    query.Set("x-vercel-protection-bypass", c.bypass)
    parsed.RawQuery = query.Encode()
    return parsed.String(), nil
}

// Register tells the server about this agent for the first time
func (c *Client) Register(ctx context.Context, referral, version string) error {
    payload := map[string]interface{}{
        "email":    c.email,
        "deviceId": c.DeviceID,
        "referral": referral,
        "version":  version,
    }
    
    response, err := c.doRequest(ctx, "POST", "/api/agent/register", payload)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }
    defer response.Body.Close()
    
    if response.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(response.Body)
        return fmt.Errorf("registration rejected: %s (status %d)", string(body), response.StatusCode)
    }
    
    return nil
}

// Beat sends a heartbeat to let the server know we're still alive
func (c *Client) Beat(ctx context.Context) error {
    payload := map[string]interface{}{
        "email":    c.email,
        "deviceId": c.DeviceID,
    }
    
    response, err := c.doRequest(ctx, "POST", "/api/agent/beat", payload)
    if err != nil {
        return fmt.Errorf("heartbeat failed: %w", err)
    }
    defer response.Body.Close()
    
    if response.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(response.Body)
        return fmt.Errorf("heartbeat rejected: %s (status %d)", string(body), response.StatusCode)
    }
    
    return nil
}

// doRequest is the internal workhorse that actually sends HTTP requests
func (c *Client) doRequest(ctx context.Context, method, path string, payload interface{}) (*http.Response, error) {
    // Build the full URL
    fullURL := c.baseURL + path

    // Add bypass parameters if we have a token
    var err error
    fullURL, err = c.addBypassParams(fullURL)
    if err != nil {
        return nil, err
    }
    
    // Convert the payload to JSON
    var body io.Reader
    if payload != nil {
        jsonData, err := json.Marshal(payload)
        if err != nil {
            return nil, fmt.Errorf("failed to encode payload: %w", err)
        }
        body = bytes.NewReader(jsonData)
    }
    
    // Create the HTTP request
    request, err := http.NewRequestWithContext(ctx, method, fullURL, body)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    // Set headers
    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("User-Agent", "IdleNet-Agent/1.0")
    
    // Add bypass header if we have a token
    if c.bypass != "" {
        request.Header.Set("x-vercel-protection-bypass", c.bypass)
    }
    
    // Send the request
    return c.httpClient.Do(request)
}

// Job represents a unit of work from the server
type Job struct {
    ID          string                 `json:"jobId"`
    Type        string                 `json:"type"`
    Args        map[string]interface{} `json:"args"`
    TimeoutSec  int                    `json:"timeoutSec"`
    ArtifactURL string                 `json:"artifactUrl"` // Now parsed from JSON
    SHA256      string                 `json:"sha256"`      // Now parsed from JSON
    MaxSeconds  int                    `json:"-"`
    MemoryMB    int                    `json:"mem_mb"`
}

// parseJobArgs sets computed fields
func (j *Job) parseJobArgs() {
    j.MaxSeconds = j.TimeoutSec
}

// GetNextJob asks the server if there's any work available
func (c *Client) GetNextJob(ctx context.Context) (*Job, error) {
    path := fmt.Sprintf("/api/agent/jobs/next?email=%s&deviceId=%s", c.email, c.DeviceID)
    resp, err := c.doRequest(ctx, "GET", path, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusNoContent {
        return nil, nil // No jobs available
    }
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var job Job
    if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
        return nil, err
    }
    job.parseJobArgs()
    return &job, nil
}

// DownloadArtifact downloads and verifies job artifact
// Supports both data URIs and regular URLs
func (c *Client) DownloadArtifact(ctx context.Context, artifactURL, expectedSHA256 string) ([]byte, error) {
    var data []byte

    // Check if it's a data URI (base64 encoded)
    if strings.HasPrefix(artifactURL, "data:") {
        // Parse data URI: data:text/javascript;base64,ZnVu...
        parts := strings.SplitN(artifactURL, ",", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid data URI format")
        }
        decoded, err := base64.StdEncoding.DecodeString(parts[1])
        if err != nil {
            return nil, fmt.Errorf("failed to decode base64: %w", err)
        }
        data = decoded
    } else {
        // Regular URL - download it with context support
        req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to create download request: %w", err)
        }

        resp, err := c.httpClient.Do(req)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        downloaded, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, err
        }
        data = downloaded
    }
    
    // Verify SHA256 if provided
    if expectedSHA256 != "" {
        hash := sha256.Sum256(data)
        actualSHA256 := hex.EncodeToString(hash[:])
        if actualSHA256 != expectedSHA256 {
            return nil, fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
        }
    }
    
    return data, nil
}

// ReportJobComplete reports job completion
func (c *Client) ReportJobComplete(ctx context.Context, jobID, status string, result map[string]interface{}, execTimeMs int64) error {
    body := map[string]interface{}{
        "jobId":         jobID,
        "status":        status,
        "result":        result,
        "executionTime": execTimeMs,
    }
    resp, err := c.doRequest(ctx, "POST", "/api/jobs/complete", body)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}

// UploadJobResult uploads a result file to the server after job completion
func (c *Client) UploadJobResult(ctx context.Context, jobID, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open result file: %w", err)
    }
    defer file.Close()

    // Create multipart form for file upload
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    part, err := writer.CreateFormFile("file", filepath.Base(filePath))
    if err != nil {
        return fmt.Errorf("failed to create form file: %w", err)
    }
    
    if _, err := io.Copy(part, file); err != nil {
        return fmt.Errorf("failed to copy file: %w", err)
    }

    if err := writer.WriteField("jobId", jobID); err != nil {
        return fmt.Errorf("failed to write form field: %w", err)
    }
    writer.Close()

    // Build URL with bypass token if available
    uploadURL := c.baseURL + "/api/jobs/upload-result"
    uploadURL, err = c.addBypassParams(uploadURL)
    if err != nil {
        return fmt.Errorf("failed to build upload URL: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
    if err != nil {
        return fmt.Errorf("failed to create upload request: %w", err)
    }
    
    req.Header.Set("Content-Type", writer.FormDataContentType())
    if c.bypass != "" {
        req.Header.Set("x-vercel-protection-bypass", c.bypass)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("upload failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("upload rejected: %s (status %d)", string(bodyBytes), resp.StatusCode)
    }

    return nil
}


