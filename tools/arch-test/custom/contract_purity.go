package custom

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ContractPurityValidator struct {
	ContractsDir string
}

func (v *ContractPurityValidator) Name() string {
	return "contract-definition-purity"
}

func (v *ContractPurityValidator) Description() string {
	return "Contract definition modules must have zero dependencies"
}

func (v *ContractPurityValidator) Check() error {
	// Check if contracts/definitions exists
	if _, err := os.Stat(v.ContractsDir); os.IsNotExist(err) {
		// No contracts directory, nothing to check
		return nil
	}

	entries, err := os.ReadDir(v.ContractsDir)
	if err != nil {
		return fmt.Errorf("failed to read contracts directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		contractPath := filepath.Join(v.ContractsDir, entry.Name())

		// Check go.mod purity
		if err := v.checkGoModPurity(contractPath, entry.Name()); err != nil {
			return err
		}

		// Check no internal imports
		if err := v.checkNoInternalImports(contractPath, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func (v *ContractPurityValidator) checkGoModPurity(contractPath, contractName string) error {
	goModPath := filepath.Join(contractPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		// No go.mod, that's fine
		return nil
	}

	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire || strings.HasPrefix(line, "require ") {
			// Skip indirect dependencies and comments
			if strings.Contains(line, "// indirect") || strings.HasPrefix(line, "//") {
				continue
			}

			// Check if it's an external dependency (contains '.')
			parts := strings.Fields(line)
			if len(parts) > 0 && strings.Contains(parts[0], ".") {
				return fmt.Errorf(
					"contracts/definitions/%s has external dependency: %s\n\n"+
						"Contract definitions must have ZERO dependencies (no require statements).\n\n"+
						"If you see service internals here, InprocServer is in the wrong location.\n"+
						"Move InprocServer to: services/%s/internal/adapters/inbound/contracts/\n\n"+
						"Contract definition modules should contain ONLY:\n"+
						"  - Interfaces (api.go)\n"+
						"  - DTOs (dto.go)\n"+
						"  - Errors (errors.go)\n"+
						"  - InprocClient (thin wrapper)",
					contractName, line, contractName,
				)
			}
		}
	}

	return nil
}

func (v *ContractPurityValidator) checkNoInternalImports(contractPath, contractName string) error {
	return filepath.Walk(contractPath, func(path string, info os.FileInfo, err error) error {
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

			if strings.Contains(importPath, "/internal/") {
				return fmt.Errorf(
					"contracts/definitions/%s/%s: imports internal package: %s\n\n"+
						"Contract definition modules must NEVER import internal/ packages from any service.\n\n"+
						"If this is InprocServer, it belongs in the service's internal adapters:\n"+
						"  Move to: services/%s/internal/adapters/inbound/contracts/inproc_server.go\n\n"+
						"Contract definition modules can only import:\n"+
						"  - Standard library\n"+
						"  - Other contract definition modules (public APIs)",
					contractName, filepath.Base(path), importPath, contractName,
				)
			}
		}

		return nil
	})
}
