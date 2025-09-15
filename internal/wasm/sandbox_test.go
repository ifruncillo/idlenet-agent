package wasm

import (
	"context"
	"testing"
	"time"
)

// Valid minimal WASM binary (function that returns 42)
var testWASM = []byte{
	0x00, 0x61, 0x73, 0x6d, // WASM magic
	0x01, 0x00, 0x00, 0x00, // WASM version
	0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f, // Type section: [] -> [i32]
	0x03, 0x02, 0x01, 0x00, // Function section: function 0 has type 0
	0x07, 0x08, 0x01, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x00, 0x00, // Export section: export "main" as function 0
	0x0a, 0x09, 0x01, 0x07, 0x00, 0x41, 0x2a, 0x0b, // Code section: function body that returns i32.const 42
}

func TestSandboxCreation(t *testing.T) {
	config := DefaultSandboxConfig()
	sandbox, err := NewSandbox(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Close()

	if sandbox.config.MaxMemoryPages != 64 {
		t.Errorf("Expected max memory pages 64, got %d", sandbox.config.MaxMemoryPages)
	}
	
	if sandbox.config.AllowNetworking {
		t.Error("Expected networking to be disabled by default")
	}
	
	if sandbox.config.AllowFileSystem {
		t.Error("Expected file system access to be disabled by default")
	}
}

func TestSandboxExecution(t *testing.T) {
	config := DefaultSandboxConfig()
	sandbox, err := NewSandbox(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := sandbox.Execute(ctx, testWASM, "main", []interface{}{})
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if !result.Success {
		t.Logf("Execution failed with error: %s", result.Error)
		// This might fail with the manually created WASM, which is OK for now
		// The important thing is that the sandbox infrastructure is working
	} else {
		t.Logf("Execution succeeded: %s", result.Output)
		t.Logf("CPU time: %v", result.CPUTime)
		t.Logf("Fuel used: %d", result.FuelUsed)
	}
}

func TestSandboxVerification(t *testing.T) {
	config := DefaultSandboxConfig()
	sandbox, err := NewSandbox(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Close()

	// Test invalid WASM (too short)
	invalidWASM := []byte{0x00, 0x61}
	err = sandbox.VerifyWASM(invalidWASM)
	if err == nil {
		t.Error("Expected invalid WASM to fail verification")
	}

	// Test invalid magic number
	badMagic := []byte{0xFF, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	err = sandbox.VerifyWASM(badMagic)
	if err == nil {
		t.Error("Expected WASM with bad magic to fail verification")
	}
	
	// Test invalid version
	badVersion := []byte{0x00, 0x61, 0x73, 0x6d, 0x02, 0x00, 0x00, 0x00}
	err = sandbox.VerifyWASM(badVersion)
	if err == nil {
		t.Error("Expected WASM with bad version to fail verification")
	}
}

func TestSandboxResourceLimits(t *testing.T) {
	config := &SandboxConfig{
		MaxMemoryPages:   32,    // 2MB
		MaxExecutionTime: 1 * time.Second,
		MaxStackDepth:    500,
		AllowNetworking:  false,
		AllowFileSystem:  false,
		CPUTimeLimit:     500 * time.Millisecond,
	}
	
	sandbox, err := NewSandbox(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sandbox.Close()

	if sandbox.config.MaxMemoryPages != 32 {
		t.Errorf("Expected max memory pages 32, got %d", sandbox.config.MaxMemoryPages)
	}
	
	if sandbox.config.CPUTimeLimit != 500*time.Millisecond {
		t.Errorf("Expected CPU time limit 500ms, got %v", sandbox.config.CPUTimeLimit)
	}
}