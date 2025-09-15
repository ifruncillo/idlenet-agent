package updater

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

// GitHubRelease represents the structure of a GitHub release
type GitHubRelease struct {
    TagName string `json:"tag_name"`
    Name    string `json:"name"`
    Assets  []struct {
        Name        string `json:"name"`
        DownloadURL string `json:"browser_download_url"`
        Size        int    `json:"size"`
    } `json:"assets"`
    PublishedAt time.Time `json:"published_at"`
}

// VersionChecker checks for new releases on GitHub
type VersionChecker struct {
    currentVersion string
    repoOwner      string
    repoName       string
    httpClient     *http.Client
}

// NewVersionChecker creates a new version checker
func NewVersionChecker(currentVersion string) *VersionChecker {
    return &VersionChecker{
        currentVersion: currentVersion,
        repoOwner:      "ifruncillo",
        repoName:       "idlenet-agent",
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// CheckForUpdate checks if a newer version is available
func (vc *VersionChecker) CheckForUpdate() (*GitHubRelease, bool, error) {
    url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", 
        vc.repoOwner, vc.repoName)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, false, err
    }
    
    // GitHub requires a user agent
    req.Header.Set("User-Agent", "IdleNet-Agent-Updater")
    
    resp, err := vc.httpClient.Do(req)
    if err != nil {
        return nil, false, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
    }
    
    var release GitHubRelease
    if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
        return nil, false, err
    }
    
    // Compare versions (remove 'v' prefix if present)
    latestVersion := strings.TrimPrefix(release.TagName, "v")
    currentVersion := strings.TrimPrefix(vc.currentVersion, "v")
    
    isNewer := vc.compareVersions(latestVersion, currentVersion) > 0
    
    return &release, isNewer, nil
}

// compareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func (vc *VersionChecker) compareVersions(v1, v2 string) int {
    // Simple string comparison for now
    // In production, you'd parse semantic versions properly
    if v1 > v2 {
        return 1
    } else if v1 < v2 {
        return -1
    }
    return 0
}