package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AoziruCake/higgsfield-action/internal/config"
	"github.com/AoziruCake/higgsfield-action/internal/runner"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	return runner.Run(ctx, cfg)
}
