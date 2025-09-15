# WASM Sandbox

This package provides secure WASM execution capabilities for the IdleNet agent.

## Features

- **Secure Execution**: WASM programs run in a sandboxed environment with strict resource limits
- **Resource Limiting**: Memory, execution time, and CPU time limits prevent resource abuse
- **Security Validation**: WASM binaries are validated before execution
- **Configurable Limits**: Sandbox limits can be adjusted based on system resources

## Security Features

- **Memory Limits**: Maximum memory pages (64KB each) to prevent excessive memory usage
- **Execution Timeouts**: Maximum execution time to prevent infinite loops
- **CPU Time Limits**: Fuel-based CPU time limiting using wasmtime's fuel mechanism
- **No File System Access**: File system access disabled by default
- **No Network Access**: Network access disabled by default
- **WASM Validation**: Basic format validation and compilation checks

## Configuration

The sandbox can be configured with custom limits:

```go
config := &wasm.SandboxConfig{
    MaxMemoryPages:   64,                 // 4MB max memory
    MaxExecutionTime: 30 * time.Second,   // 30 second timeout
    MaxStackDepth:    1000,               // Call stack depth limit
    AllowNetworking:  false,              // No network access
    AllowFileSystem:  false,              // No file system access
    CPUTimeLimit:     10 * time.Second,   // CPU time limit
}
```

## Usage

```go
// Create sandbox
sandbox, err := wasm.NewSandbox(wasm.DefaultSandboxConfig())
if err != nil {
    return err
}
defer sandbox.Close()

// Verify WASM binary
if err := sandbox.VerifyWASM(wasmBytes); err != nil {
    return fmt.Errorf("WASM validation failed: %w", err)
}

// Execute WASM
result, err := sandbox.Execute(ctx, wasmBytes, "main", []interface{}{})
if err != nil {
    return err
}

if result.Success {
    fmt.Printf("Execution successful: %s\n", result.Output)
} else {
    fmt.Printf("Execution failed: %s\n", result.Error)
}
```

## Resource Monitoring

The sandbox tracks resource usage during execution:

- **CPU Time**: Actual execution time
- **Fuel Usage**: wasmtime fuel consumption (rough CPU usage metric)
- **Memory**: Peak memory usage during execution
- **Success/Failure**: Execution outcome with detailed error messages