package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sync"

	"multi-ping/internal/ping"
	"multi-ping/internal/ui"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, err := ParseConfig(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	updates := make(chan ping.Update, len(cfg.Checks)*4)
	renderer := ui.NewRenderer(stdout, ui.Config{
		Checks:   cfg.Checks,
		Interval: cfg.Interval,
		Timeout:  cfg.Timeout,
	})

	var wg sync.WaitGroup
	for _, check := range cfg.Checks {
		runner := ping.Runner{
			CheckKey: check,
			Interval: cfg.Interval,
			Timeout:  cfg.Timeout,
			Out:      updates,
		}
		wg.Add(1)
		go runner.Run(ctx, &wg)
	}

	renderErr := renderer.Run(ctx, updates)
	wg.Wait()

	if renderErr != nil {
		return fmt.Errorf("render dashboard: %w", renderErr)
	}

	return nil
}
