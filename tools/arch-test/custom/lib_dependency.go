package custom

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type LibDependencyValidator struct {
	LibsDir  string
	RepoRoot string
}

func (v *LibDependencyValidator) Name() string {
	return "lib-dependency-purity"
}

func (v *LibDependencyValidator) Description() string {
	return "libs/ packages can only import stdlib, external deps, or other libs"
}

func (v *LibDependencyValidator) Check() error {
	// Check if libs directory exists
	if _, err := os.Stat(v.LibsDir); os.IsNotExist(err) {
		return nil
	}

	rootModuleName, err := v.getRootModuleName()
	if err != nil {
		return fmt.Errorf("failed to get root module name: %w", err)
	}

	return filepath.Walk(v.LibsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if it's importing from the same module
			if strings.HasPrefix(importPath, rootModuleName+"/") {
				relPath := strings.TrimPrefix(importPath, rootModuleName+"/")
				firstPart := strings.Split(relPath, "/")[0]

				// Only allow libs/ imports from same module
				if firstPart != "libs" {
					return fmt.Errorf(
						"%s: lib imports forbidden package: %s\n\n"+
							"libs/ packages can only import:\n"+
							"  - Standard library\n"+
							"  - External dependencies\n"+
							"  - Other libs/ packages\n\n"+
							"Forbidden: services/, contracts/, tools/, or root packages",
						path, importPath,
					)
				}
			}
		}

		return nil
	})
}

func (v *LibDependencyValidator) getRootModuleName() (string, error) {
	goModPath := filepath.Join(v.RepoRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module name not found in %s", goModPath)
}
