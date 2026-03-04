package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPurityValidator_ZeroDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0755)

	// Create go.mod with external dependency
	goMod := `module github.com/test/testcontract

go 1.21

require (
	github.com/external/pkg v1.0.0
)
`
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for contract with external dependencies")
	}

	if !strings.Contains(err.Error(), "external dependency") {
		t.Errorf("Expected error about external dependency, got: %v", err)
	}
}

func TestContractPurityValidator_NoInternalImports(t *testing.T) {
	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0755)

	// Create go.mod with no dependencies
	goMod := `module github.com/test/testcontract

go 1.21
`
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0644)

	// Create .go file with internal import
	goFile := `package testcontract

import (
	"github.com/test/services/todo/internal/domain"
)

type Client struct {}
`
	os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(goFile), 0644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for contract importing internal package")
	}

	if !strings.Contains(err.Error(), "/internal/") {
		t.Errorf("Expected error about internal import, got: %v", err)
	}
}

func TestContractPurityValidator_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0755)

	// Create clean go.mod
	goMod := `module github.com/test/testcontract

go 1.21
`
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0644)

	// Create .go file with only stdlib imports
	goFile := `package testcontract

import (
	"context"
	"errors"
)

type Client struct {}
`
	os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(goFile), 0644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for valid contract, got: %v", err)
	}
}
