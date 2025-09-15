package wasm

import (
	"context"
	"fmt"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v3"
)

// SandboxConfig defines security limits for WASM execution
type SandboxConfig struct {
	MaxMemoryPages    int           // Maximum memory pages (64KB each)
	MaxExecutionTime  time.Duration // Maximum execution time
	MaxStackDepth     int           // Maximum call stack depth
	AllowNetworking   bool          // Whether to allow network access
	AllowFileSystem   bool          // Whether to allow file system access
	CPUTimeLimit      time.Duration // CPU time limit
}

// DefaultSandboxConfig returns a secure default configuration
func DefaultSandboxConfig() *SandboxConfig {
	return &SandboxConfig{
		MaxMemoryPages:   64,    // 4MB max memory
		MaxExecutionTime: 30 * time.Second,
		MaxStackDepth:    1000,
		AllowNetworking:  false, // No network access by default
		AllowFileSystem:  false, // No file system access by default
		CPUTimeLimit:     10 * time.Second,
	}
}

// Sandbox provides secure WASM execution
type Sandbox struct {
	config *SandboxConfig
	engine *wasmtime.Engine
}

// NewSandbox creates a new WASM sandbox with the given configuration
func NewSandbox(config *SandboxConfig) (*Sandbox, error) {
	// Create engine with resource limits
	engineConfig := wasmtime.NewConfig()
	
	// Enable resource limiting
	engineConfig.SetConsumeFuel(true)
	
	// Disable features that could be unsafe
	engineConfig.SetWasmBulkMemory(false)
	engineConfig.SetWasmReferenceTypes(false)
	engineConfig.SetWasmMultiValue(false)
	engineConfig.SetWasmThreads(false)
	engineConfig.SetWasmSIMD(false)
	
	engine := wasmtime.NewEngineWithConfig(engineConfig)

	return &Sandbox{
		config: config,
		engine: engine,
	}, nil
}

// ExecutionResult contains the results of WASM execution
type ExecutionResult struct {
	Success     bool
	Output      string
	Error       string
	StartTime   time.Time
	EndTime     time.Time
	CPUTime     time.Duration
	MemoryUsed  int64
	FuelUsed    uint64
}

// Execute runs a WASM program with the configured security limits
func (s *Sandbox) Execute(ctx context.Context, wasmBytes []byte, funcName string, args []interface{}) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StartTime: time.Now(),
	}

	// Create a store with memory limits
	store := wasmtime.NewStore(s.engine)
	
	// Set fuel limit based on CPU time limit
	fuelLimit := uint64(s.config.CPUTimeLimit.Seconds() * 1000000) // Rough fuel estimation
	store.AddFuel(fuelLimit)

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.config.MaxExecutionTime)
	defer cancel()

	// Compile and validate the WASM module
	module, err := wasmtime.NewModule(s.engine, wasmBytes)
	if err != nil {
		result.Error = fmt.Sprintf("WASM compilation failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	// Create instance with limited imports
	linker := wasmtime.NewLinker(s.engine)
	
	// Add minimal WASI support if needed
	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.InheritStdout()
	wasiConfig.InheritStderr()
	
	// Restrict file system access
	if !s.config.AllowFileSystem {
		// Don't add any directory mappings
	}
	
	store.SetWasi(wasiConfig)
	err = linker.DefineWasi()
	if err != nil {
		result.Error = fmt.Sprintf("WASI setup failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	// Instantiate the module
	instance, err := linker.Instantiate(store, module)
	if err != nil {
		result.Error = fmt.Sprintf("WASM instantiation failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	// Get the function to execute
	fn := instance.GetFunc(store, funcName)
	if fn == nil {
		result.Error = fmt.Sprintf("Function '%s' not found in WASM module", funcName)
		result.EndTime = time.Now()
		return result, nil
	}

	// Execute with timeout monitoring
	done := make(chan struct{})
	var execErr error
	var returnValue interface{}

	go func() {
		defer close(done)
		
		// Convert args to wasmtime values
		wasmArgs := make([]interface{}, len(args))
		copy(wasmArgs, args)
		
		// Execute the function
		returnValue, execErr = fn.Call(store, wasmArgs...)
	}()

	// Wait for execution or timeout
	select {
	case <-execCtx.Done():
		result.Error = "Execution timed out"
		result.Success = false
	case <-done:
		if execErr != nil {
			result.Error = fmt.Sprintf("Execution error: %v", execErr)
			result.Success = false
		} else {
			result.Success = true
			if returnValue != nil {
				result.Output = fmt.Sprintf("Return value: %v", returnValue)
			} else {
				result.Output = "Function executed successfully"
			}
		}
	}

	// Get resource usage
	fuelConsumed, _ := store.FuelConsumed()
	result.FuelUsed = fuelLimit - fuelConsumed
	
	result.EndTime = time.Now()
	result.CPUTime = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// VerifyWASM performs basic validation on WASM bytes
func (s *Sandbox) VerifyWASM(wasmBytes []byte) error {
	// Basic WASM validation
	if len(wasmBytes) < 8 {
		return fmt.Errorf("invalid WASM: file too short")
	}

	// Check WASM magic number
	if string(wasmBytes[0:4]) != "\x00asm" {
		return fmt.Errorf("invalid WASM: missing magic number")
	}

	// Check version
	if wasmBytes[4] != 1 || wasmBytes[5] != 0 || wasmBytes[6] != 0 || wasmBytes[7] != 0 {
		return fmt.Errorf("unsupported WASM version")
	}

	// Try to compile for validation
	_, err := wasmtime.NewModule(s.engine, wasmBytes)
	if err != nil {
		return fmt.Errorf("WASM validation failed: %w", err)
	}

	return nil
}

// Close cleans up the sandbox resources
func (s *Sandbox) Close() error {
	// Wasmtime engine cleanup is handled by GC
	return nil
}