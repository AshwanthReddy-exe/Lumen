package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AshwanthReddy-exe/Lumen/internal/setup"
)

const actionRequired = setup.ActionRequired

func runForTest(args []string) setup.Report { return run(args) }

func run(args []string) setup.Report {
	if len(args) == 1 {
		switch args[0] {
		case "connect":
			return setup.Report{Outcome: setup.ActionRequired, AvailableInBlock: 3}
		case "setup":
			return placeholder("setup_not_implemented")
		case "doctor":
			return placeholder("doctor_not_implemented")
		}
	}
	if len(args) == 2 && args[0] == "service" {
		switch args[1] {
		case "start", "stop", "restart", "status":
			return placeholder("service_not_implemented")
		}
	}
	return placeholder("invalid_command")
}

func placeholder(code string) setup.Report {
	return setup.Report{Outcome: setup.ActionRequired, Actions: []setup.Action{{Code: code}}}
}

func main() {
	report := run(os.Args[1:])
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
