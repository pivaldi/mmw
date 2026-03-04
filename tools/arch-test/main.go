package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pivaldi/mmw/tools/arch-test/custom"
	"github.com/pivaldi/mmw/tools/arch-test/orchestrator"
	"github.com/pivaldi/mmw/tools/arch-test/reporter"
)

const archTaskName = "arch:test"

var headerColor = color.New(color.FgBlue)
var errorColor = color.New(color.FgRed)

func main() {
	// Discover all services
	services, err := orchestrator.DiscoverServices("./services", archTaskName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering services: %v\n", err)
		os.Exit(1)
	}

	if len(services) == 0 {
		fmt.Println("No services found to validate")
		os.Exit(0)
	}

	rep := reporter.NewReporter(os.Stdout)
	rep.PrintHeader("Architecture Validation")

	// Run arch checks for each service
	headerColor.Println("Running architecture checks…")
	result := orchestrator.CheckResult{}
	for _, service := range services {
		var checkErr error
		if service.HasArchCheck {
			result = orchestrator.RunServiceCheck(service.Path, service.Name)
			if result.ExitCode != 0 {
				eCodeStr := strconv.Itoa(result.ExitCode)
				outPut := errorColor.Sprint(strings.TrimSpace(result.Output))
				checkErr = errors.New(outPut + errorColor.Sprint("exit code "+eCodeStr))
			}
		} else {
			checkErr = fmt.Errorf("no mise task '%s' detected", archTaskName)
		}

		rep.PrintCheck(
			service.Name,
			"Validating service architecture boundaries",
			checkErr,
		)
	}

	// Run custom validators
	headerColor.Println("Running custom validators…")

	// Contract purity validator
	contractValidator := &custom.ContractPurityValidator{
		ContractsDir: "./contracts/definitions",
	}
	err = contractValidator.Check()
	rep.PrintCheck(
		contractValidator.Name(),
		contractValidator.Description(),
		err,
	)

	// Library dependency validator
	libValidator := &custom.LibDependencyValidator{
		LibsDir:  "./libs",
		RepoRoot: ".",
	}
	err = libValidator.Check()
	rep.PrintCheck(
		libValidator.Name(),
		libValidator.Description(),
		err,
	)

	exitCode := rep.Summary()
	os.Exit(exitCode)
}
