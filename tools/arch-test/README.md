# arch-test: Architecture Testing Tool

A comprehensive architecture testing tool that validates Clean Architecture principles for Go services in the monorepo.

## Overview

`arch-test` combines two types of architecture validation:

1. **arch-go validation**: Uses the [arch-go](https://github.com/fdaines/arch-go) tool to enforce dependency rules defined in `.arch-go.yml`
2. **Custom shell script validation**: Provides additional monorepo-specific checks

## Usage

### From Monorepo Root

Run architecture tests for all services:

```bash
./poc/tools/arch-test/arch-test.sh
```

Run for a specific service:

```bash
./poc/tools/arch-test/arch-test.sh todo
```

### From Service Directory

Run architecture tests for the current service:

```bash
cd poc/services/todo
../../tools/arch-test/arch-test.sh
```

## What It Checks

### arch-go Checks (via .arch-go.yml)

The `.arch-go.yml` configuration enforces Clean Architecture principles:

1. **Service Isolation**
   - Services cannot import from other services
   - Prevents tight coupling between services

2. **Domain Layer Purity**
   - Domain layer has no external dependencies
   - No HTTP, database, or infrastructure imports
   - Pure business logic only

3. **Application Layer Rules**
   - Application layer can only depend on domain
   - No adapter or infrastructure dependencies
   - Use cases remain portable

4. **Adapter Direction**
   - Primary adapters (handlers) depend on application/domain
   - Secondary adapters (repositories) depend on domain interfaces
   - Dependency inversion principle enforced

5. **Layer Dependencies**
   - Domain: No dependencies (innermost layer)
   - Application: Depends only on domain
   - Adapters: Depend on domain/application
   - Infrastructure: Wires everything together (outermost layer)

6. **Content Rules**
   - Prevents domain from importing HTTP, SQL, ORM packages
   - Prevents application from importing web frameworks
   - Ensures architectural boundaries are respected

### Custom Shell Script Checks

The `arch-test.sh` script provides additional validations:

1. **Project Structure Validation**
   - Verifies required directories exist (domain, application, adapters)
   - Ensures consistent project layout across services

2. **Service Isolation**
   - Prevents cross-service imports
   - Validates import paths stay within service boundaries

3. **Dependency Direction**
   - Checks that inner layers don't depend on outer layers
   - Validates Clean Architecture dependency flow

4. **Configuration Validation**
   - Ensures `.arch-go.yml` exists and is valid
   - Checks for required architectural rules

## Setup for New Services

When creating a new Go service:

1. **Copy the arch-go template:**
   ```bash
   cp poc/tools/arch-test/.arch-go.yml.template poc/services/your-service/.arch-go.yml
   ```

2. **Customize for your service (optional):**
   - Edit `.arch-go.yml` to add service-specific rules
   - Adjust package patterns to match your structure
   - Add custom content or naming rules

3. **Create the required directory structure:**
   ```bash
   cd poc/services/your-service
   mkdir -p domain application adapters/primary adapters/secondary infrastructure
   ```

4. **Run the architecture tests:**
   ```bash
   ../../tools/arch-test/arch-test.sh
   ```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run Architecture Tests
  run: |
    ./poc/tools/arch-test/arch-test.sh
```

### GitLab CI

```yaml
arch-test:
  script:
    - ./poc/tools/arch-test/arch-test.sh
  rules:
    - changes:
        - poc/services/**/*.go
```

### Jenkins

```groovy
stage('Architecture Tests') {
    steps {
        sh './poc/tools/arch-test/arch-test.sh'
    }
}
```

## Architecture Principles Enforced

### Clean Architecture Layers

```
┌─────────────────────────────────────────┐
│         Infrastructure (Outermost)       │  ← Wiring, DI, main.go
├─────────────────────────────────────────┤
│              Adapters                    │  ← HTTP handlers, DB repos
├─────────────────────────────────────────┤
│            Application                   │  ← Use cases, business operations
├─────────────────────────────────────────┤
│         Domain (Innermost, Pure)         │  ← Business entities, rules
└─────────────────────────────────────────┘
```

**Dependency Rule**: Dependencies point inward only. Inner layers never depend on outer layers.

### Benefits

1. **Testability**: Pure domain and application layers are easy to test
2. **Maintainability**: Clear boundaries make code easier to understand
3. **Flexibility**: Can swap out adapters without affecting business logic
4. **Independence**: Business logic doesn't depend on frameworks or databases

## Troubleshooting

### arch-go not found

Install arch-go:
```bash
go install github.com/fdaines/arch-go@latest
```

### Configuration errors

Validate your `.arch-go.yml`:
```bash
arch-go describe
```

### Test failures

Review the error messages to identify:
- Which layer is violating dependency rules
- What packages are being imported incorrectly
- Which files need refactoring

## Examples

### Valid Architecture

```go
// domain/task.go - Pure business logic
package domain

type Task struct {
    ID          string
    Description string
    Completed   bool
}

type TaskRepository interface {
    Save(task Task) error
    FindByID(id string) (Task, error)
}
```

```go
// application/task_service.go - Use case
package application

import "your-service/domain"

type TaskService struct {
    repo domain.TaskRepository
}

func (s *TaskService) CompleteTask(id string) error {
    task, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }
    task.Completed = true
    return s.repo.Save(task)
}
```

```go
// adapters/secondary/postgres_task_repository.go - Adapter
package secondary

import (
    "database/sql"
    "your-service/domain"
)

type PostgresTaskRepository struct {
    db *sql.DB
}

func (r *PostgresTaskRepository) Save(task domain.Task) error {
    // SQL implementation
}
```

### Invalid Architecture (Will Fail Tests)

```go
// domain/task.go - INVALID: Domain depending on database
package domain

import "database/sql" // ❌ Violation: Domain importing infrastructure

type Task struct {
    db *sql.DB // ❌ Violation: Infrastructure in domain
}
```

```go
// application/task_service.go - INVALID: Application depending on adapter
package application

import "your-service/adapters/secondary" // ❌ Violation: Wrong dependency direction

type TaskService struct {
    repo *secondary.PostgresTaskRepository // ❌ Should use domain interface
}
```

## Related Tools

- [arch-go](https://github.com/fdaines/arch-go) - Go architecture testing framework
- [go-cleanarch](https://github.com/roblaszczak/go-cleanarch) - Alternative architecture validator

## Contributing

When adding new architectural rules:

1. Update `.arch-go.yml.template` with new rules
2. Document the rules in this README
3. Add test cases to verify enforcement
4. Update CI/CD examples if needed

## License

See the monorepo root LICENSE file.
