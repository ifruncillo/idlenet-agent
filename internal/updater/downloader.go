package updater

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
)

// Downloader handles downloading and verifying updates
type Downloader struct {
    httpClient *http.Client
    tempDir    string
}

// NewDownloader creates a new downloader
func NewDownloader() (*Downloader, error) {
    tempDir := filepath.Join(os.TempDir(), "idlenet-updates")
    if err := os.MkdirAll(tempDir, 0755); err != nil {
        return nil, err
    }
    
    return &Downloader{
        httpClient: &http.Client{},
        tempDir:    tempDir,
    }, nil
}

// DownloadUpdate downloads the appropriate binary for this platform
func (d *Downloader) DownloadUpdate(release *GitHubRelease) (string, error) {
    // Determine the correct asset name for this platform
    assetName := d.getAssetName()
    
    // Find the matching asset
    var downloadURL string
    for _, asset := range release.Assets {
        if asset.Name == assetName {
            downloadURL = asset.DownloadURL
            break
        }
    }
    
    if downloadURL == "" {
        return "", fmt.Errorf("no release found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
    }
    
    // Download to temp file
    tempFile := filepath.Join(d.tempDir, assetName)
    
    resp, err := d.httpClient.Get(downloadURL)
    if err != nil {
        return "", fmt.Errorf("download failed: %w", err)
    }
    defer resp.Body.Close()
    
    out, err := os.Create(tempFile)
    if err != nil {
        return "", fmt.Errorf("failed to create temp file: %w", err)
    }
    defer out.Close()
    
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to save update: %w", err)
    }
    
    return tempFile, nil
}

// getAssetName returns the expected asset name for this platform
func (d *Downloader) getAssetName() string {
    name := fmt.Sprintf("idlenet-%s-%s", runtime.GOOS, runtime.GOARCH)
    if runtime.GOOS == "windows" {
        name += ".exe"
    }
    return name
}

// VerifyChecksum verifies the SHA256 checksum of a file
func (d *Downloader) VerifyChecksum(filepath, expectedChecksum string) error {
    file, err := os.Open(filepath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return err
    }
    
    actualChecksum := hex.EncodeToString(hasher.Sum(nil))
    if actualChecksum != expectedChecksum {
        return fmt.Errorf("checksum mismatch: expected %s, got %s", 
            expectedChecksum, actualChecksum)
    }
    
    return nil
}

// CleanupTemp removes temporary download files
func (d *Downloader) CleanupTemp() error {
    return os.RemoveAll(d.tempDir)
}