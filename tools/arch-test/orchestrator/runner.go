package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CheckResult represents the result of running arch:check on a service
type CheckResult struct {
	ServiceName string
	ExitCode    int
	Output      string
}

// RunServiceCheck executes mise run arch:check for a service
func RunServiceCheck(servicePath, serviceName string) CheckResult {
	// First, trust the mise.toml file in this directory
	trustCmd := exec.CommandContext(context.Background(), "mise", "trust")
	trustCmd.Dir = servicePath
	if err := trustCmd.Run(); err != nil {
		panic(fmt.Errorf("mise.toml is not trusted: %w", err))
	}

	// Now run the arch:check task
	cmd := exec.CommandContext(context.Background(), "mise", "run", "arch:check")
	cmd.Dir = servicePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			exitCode = 1
		}
	}

	// Combine stdout and stderr for output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	return CheckResult{
		ServiceName: serviceName,
		ExitCode:    exitCode,
		Output:      output,
	}
}
