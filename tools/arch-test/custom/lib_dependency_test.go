package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibDependencyValidator_NoRootDirImports(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "libs", "mylib")
	os.MkdirAll(libDir, 0755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0644)

	// Create lib file importing services (forbidden)
	libFile := `package mylib

import (
	"github.com/test/project/services/todo"
)

func DoSomething() {}
`
	os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(libFile), 0644)

	validator := &LibDependencyValidator{
		LibsDir:    filepath.Join(tmpDir, "libs"),
		RepoRoot:   tmpDir,
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for lib importing services/")
	}

	if !strings.Contains(err.Error(), "services/") {
		t.Errorf("Expected error about services/ import, got: %v", err)
	}
}

func TestLibDependencyValidator_AllowsStdlibAndExternal(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "libs", "mylib")
	os.MkdirAll(libDir, 0755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0644)

	// Create lib file importing stdlib and external deps
	libFile := `package mylib

import (
	"context"
	"fmt"
	"github.com/external/package"
)

func DoSomething() {}
`
	os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(libFile), 0644)

	validator := &LibDependencyValidator{
		LibsDir:    filepath.Join(tmpDir, "libs"),
		RepoRoot:   tmpDir,
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for stdlib and external imports, got: %v", err)
	}
}

func TestLibDependencyValidator_AllowsOtherLibs(t *testing.T) {
	tmpDir := t.TempDir()
	libADir := filepath.Join(tmpDir, "libs", "liba")
	libBDir := filepath.Join(tmpDir, "libs", "libb")
	os.MkdirAll(libADir, 0755)
	os.MkdirAll(libBDir, 0755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0644)

	// Create lib A
	libAFile := `package liba

func FuncA() {}
`
	os.WriteFile(filepath.Join(libADir, "liba.go"), []byte(libAFile), 0644)

	// Create lib B importing lib A (allowed)
	libBFile := `package libb

import (
	"github.com/test/project/libs/liba"
)

func FuncB() {
	liba.FuncA()
}
`
	os.WriteFile(filepath.Join(libBDir, "libb.go"), []byte(libBFile), 0644)

	validator := &LibDependencyValidator{
		LibsDir:    filepath.Join(tmpDir, "libs"),
		RepoRoot:   tmpDir,
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for lib importing other lib, got: %v", err)
	}
}
