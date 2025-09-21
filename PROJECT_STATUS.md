# IdleNet Agent - Project Status & Architecture
Last Updated: September 21, 2025
Current Version: v1.0.0
Repository: https://github.com/ifruncillo/idlenet-agent

## Executive Summary
IdleNet is a distributed computing marketplace that allows users to monetize their idle computing resources by running WebAssembly jobs in a sandboxed environment. The agent runs as a background service, detecting when the computer is idle and safely executing computational tasks for paying customers.

## COMPLETED FEATURES (What Already Exists - DO NOT REIMPLEMENT)

### Core Architecture
- Modular Go application with clean separation of concerns
- Main orchestrator at cmd/idlenet/main.go coordinates all components
- Internal packages for each major subsystem (see Architecture section)

### Key Functional Components
- WASM Execution Engine (internal/executor/)
  - Sandboxed WebAssembly runtime using wasmtime-go
  - Safe execution of untrusted customer code
  - Resource limits and timeout enforcement
  
- Windows Installer (IdleNet-Setup-1.0.0.exe)
  - NSIS-based installer with setup wizard
  - First-run configuration flow
  - Service installation capability
  
- System Tray UI (internal/ui/)
  - System tray icon with menu
  - Settings interface (HTML-based)
  - Real-time status display
  
- Idle Detection (internal/idle/)
  - Cross-platform support (Windows/Linux/Mac)
  - Configurable idle threshold
  - Respects user activity
  
- Metrics & Performance Tracking (internal/metrics/)
  - Job completion statistics
  - CPU time tracking
  - Earnings calculation
  - Session stats on shutdown
  
- Auto-Updater (internal/updater/)
  - Checks GitHub releases for new versions
  - Self-updating capability
  - Version comparison logic
  
- Resource Management (internal/resource/)
  - CPU/memory limit enforcement
  - Configurable resource modes
  - Safe resource allocation
  
- API Communication (internal/api/)
  - Heartbeat system (30-second intervals)
  - Job polling (20-second intervals)
  - Vercel deployment integration
  - Error handling and retry logic

### Configuration System (internal/config/)
- Unified configuration management
- Device ID generation using UUID
- Email and referral code handling
- Cross-platform config file locations

## Directory Structure
/idlenet-agent
  /cmd/idlenet/           - Main application entry point
    main.go              - Orchestrator (coordinates components)
  /internal/             - Private application packages
    /api/                - Server communication client
    /config/             - Configuration management
    /executor/           - WASM job execution engine
    /idle/               - Idle detection (cross-platform)
    /metrics/            - Performance tracking & stats
    /onboarding/         - First-run setup wizard
    /resource/           - CPU/memory management
    /ui/                 - System tray & settings UI
    /updater/            - Auto-update functionality
  /.github/workflows/    - GitHub Actions CI/CD
  build-idlenet.ps1      - Build automation script
  installer.nsi          - Windows installer script
  go.mod                 - Go dependencies
  go.sum                 - Dependency checksums

## Current Working State
- Agent successfully connects to production API at https://idlenet-pilot-qi7t.vercel.app
- Heartbeat and job polling loops functioning
- Configuration system unified and working
- All v1.0.0 features integrated and building successfully

## Deployment & Distribution
- Binaries Available: Windows (amd64), Linux (amd64), Darwin (amd64/arm64)
- Windows Installer: IdleNet-Setup-1.0.0.exe with setup wizard
- GitHub Releases: Automated release workflow configured

## Known Issues & Limitations
- WASM execution currently in testing phase
- Linux WASM support needed additional fixes (completed)
- Job execution from API needs real job types beyond test "sleep" and "hash"

## Immediate Priorities (Next Steps)
1. Test complete integrated system end-to-end
2. Implement real WASM job types from customer workloads
3. Add Stripe Connect for payment processing
4. Build customer dashboard for job submission
5. Enhance metrics reporting and analytics

## Technical Decisions & Constraints
- Go 1.22 required
- Uses wasmtime-go v15 for WebAssembly runtime
- System tray uses getlantern/systray library
- Targets 64-bit architectures primarily
- Config stored in platform-specific locations (ProgramData on Windows)

## Development Setup
Clone repository:
  git clone https://github.com/ifruncillo/idlenet-agent.git
  cd idlenet-agent

Install dependencies:
  go mod tidy

Build the agent:
  go build -o idlenet.exe ./cmd/idlenet

Run the agent:
  ./idlenet.exe

## Important Notes for Contributors
- DO NOT recreate features listed in Completed Features section
- All new features should be added as packages under internal/
- Maintain modular architecture - no monolithic code in main.go
- Use existing packages before creating new ones
- Check git history for context on architectural decisions

## API Endpoints (Production)
- Base URL: https://idlenet-pilot-qi7t.vercel.app
- POST /api/agent/register - Agent registration
- POST /api/agent/beat - Heartbeat
- GET /api/agent/jobs/next - Poll for jobs
- POST /api/agent/jobs/report - Report job results
