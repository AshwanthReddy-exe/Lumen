package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AshwanthReddy-exe/Lumen/internal/control"
	"github.com/AshwanthReddy-exe/Lumen/internal/host"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	if len(args) == 0 {
		return usage()
	}
	c, err := host.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration unavailable")
		return 3
	}
	switch args[0] {
	case "init":
		if len(args) != 1 {
			return usage()
		}
		if err := host.Initialize(c); err != nil {
			fmt.Fprintln(os.Stderr, "initialization unavailable")
			return 3
		}
		return 0
	case "serve":
		if len(args) != 1 {
			return usage()
		}
		return serve(c)
	case "status", "shutdown":
		if len(args) != 1 {
			return usage()
		}
		return call(c, args[0])
	default:
		if len(args) == 2 && args[0] == "task" && (args[1] == "submit" || args[1] == "show" || args[1] == "cancel") {
			return call(c, args[0]+" "+args[1])
		}
		if len(args) == 2 && args[0] == "approval" && args[1] == "resolve" {
			return call(c, args[0]+" "+args[1])
		}
		return usage()
	}
}
func usage() int {
	fmt.Fprintln(os.Stderr, "usage: lumen-host init|serve|status|task submit|task show|task cancel|approval resolve|shutdown")
	return 2
}
func serve(c host.Config) int {
	s, err := host.New(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "state unavailable")
		return 3
	}
	if err := s.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "service unavailable")
		return 3
	}
	fmt.Println("ready")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); s.Shutdown() }()
	_ = s.Wait(context.Background())
	return 0
}
func call(c host.Config, cmd string) int {
	r, err := control.Call(c.SocketPath, c.CredentialPath, control.Request{Command: cmd})
	if err != nil {
		fmt.Fprintln(os.Stderr, "host unavailable")
		return 3
	}
	if !r.OK || r.Error != "" {
		fmt.Fprintln(os.Stderr, r.Error)
		return 3
	}
	if r.Data != nil {
		fmt.Println(r.Data)
	}
	return 0
}
