---
name: ovya-guideline-go
description: Apply OVYA Go syntax guidelines. Use when user ask for ovya Go code guidelines.
---

- Wrap external errors:
  — never return raw errors from external packages
  - use `fmt.Errorf("doing X: %w", err)` for domain or library package
  - use `eris.Wrap(err, "doing X")` for others packages
- Ensure there is always a blank line before return.
- When modifying Go code, ensure the final result pass the command `golangci-lint run`
