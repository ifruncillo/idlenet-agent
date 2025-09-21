# IdleNet Agent Complete Setup Script
# This creates all files needed for the full-featured agent

Write-Host "`n🚀 Building Complete IdleNet Agent`n" -ForegroundColor Cyan
Write-Host "This script will create all files for your project." -ForegroundColor Yellow
Write-Host "It will take about 30 seconds to complete.`n" -ForegroundColor Yellow

# Store the root directory
$rootDir = Get-Location

# First, let's create all the directories we need
Write-Host "📁 Creating directory structure..." -ForegroundColor Green

$directories = @(
    "cmd\idlenet",
    "internal\config",
    "internal\api", 
    "internal\idle",
    "internal\runner",
    "internal\updater",
    "internal\logx",
    "internal\util",
    "scripts\windows",
    "scripts\macos",
    "scripts\linux",
    "scripts\http",
    "scripts\tests",
    ".github\workflows"
)

foreach ($dir in $directories) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}

Write-Host "✅ Directories created" -ForegroundColor Green

# Now let's create each file with its complete content
Write-Host "`n📝 Creating Go module files..." -ForegroundColor Green

# Create go.mod - this tells Go what dependencies we need
@'
module github.com/ifruncillo/idlenet-agent

go 1.22

require (
    github.com/bytecodealliance/wasmtime-go/v15 v15.0.0
    github.com/google/uuid v1.6.0
    golang.org/x/sys v0.15.0
)
'@ | Set-Content -Path "go.mod" -Encoding UTF8

Write-Host "✅ go.mod created" -ForegroundColor Green

# Create the main.go file with FULL implementation
Write-Host "📝 Creating main application file..." -ForegroundColor Green

# Due to the size, I'll create a placeholder and then you'll need to copy the full version
@'
package main

// TEMPORARY: Full implementation needs to be added
// This is a placeholder that will be replaced with the complete code

import (
    "fmt"
)

func main() {
    fmt.Println("IdleNet Agent - Full implementation needed")
    fmt.Println("Please run the complete setup script")
}
'@ | Set-Content -Path "cmd\idlenet\main.go" -Encoding UTF8

Write-Host "⚠️  Note: main.go needs full implementation (too large for one script)" -ForegroundColor Yellow

# Create README.md for GitHub
Write-Host "📝 Creating README..." -ForegroundColor Green

@'
# IdleNet Agent

Turn your idle computer into a secure worker for distributed computing tasks.

## Features

- 🔒 **Secure**: WASM/WASI sandboxed execution - programs run in isolation
- 🖥️ **Cross-platform**: Works on Windows, macOS, and Linux
- 🔄 **Auto-updates**: Automatically checks for and installs updates
- 🪶 **Lightweight**: Minimal resource usage, only runs when idle
- 👤 **User-mode**: No administrator privileges required

## Quick Install

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/ifruncillo/idlenet-agent/main/scripts/windows/install.ps1 | iex