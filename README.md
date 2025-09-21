# IdleNet Agent

Turn your idle computer into a secure worker for distributed computing tasks.

## Quick Install

### Windows (PowerShell as Administrator)
```powershell
irm https://raw.githubusercontent.com/ifruncillo/idlenet-agent/main/scripts/windows/install.ps1 | iex
```

### macOS
```bash
curl -sSL https://raw.githubusercontent.com/ifruncillo/idlenet-agent/main/scripts/macos/install.sh | bash
```

### Linux
```bash
curl -sSL https://raw.githubusercontent.com/ifruncillo/idlenet-agent/main/scripts/linux/install.sh | bash
```

## Features

- ?? **Secure**: WASM/WASI sandboxed execution
- ??? **Cross-platform**: Windows, macOS, Linux
- ?? **Auto-updates**: Automatic version checking
- ?? **Lightweight**: Minimal resource usage
- ?? **User-mode**: No admin privileges required

## Configuration

Set before first run:
- `IDLENET_EMAIL` - Your email address
- `IDLENET_REF` - Referral code (optional)
- `IDLENET_API_BASE` - API endpoint (optional)

## Build from Source

```bash
go build -o idlenet ./cmd/idlenet
```

## License

MIT
