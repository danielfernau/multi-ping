package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"multi-ping/internal/app"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "multi-ping: %v\n", err)
		os.Exit(1)
	}
}
