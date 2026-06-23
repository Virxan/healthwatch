// Command healthwatch is a small, dependency-light website health-check
// aggregator: it periodically checks a list of HTTP(S) targets for
// reachability, latency and TLS certificate expiry, and serves the
// results over a JSON API and a minimal dashboard.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"healthwatch/internal/api"
	"healthwatch/internal/checker"
	"healthwatch/internal/config"
	"healthwatch/internal/scheduler"
	"healthwatch/internal/store"
)

func main() {
	configPath := flag.String("config", "config/targets.yaml", "path to the targets YAML file")
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(*configPath, *addr, logger); err != nil {
		logger.Error("healthwatch exited with an error", "error", err)
		os.Exit(1)
	}
}

func run(configPath, addr string, logger *slog.Logger) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	st := store.NewMemory()
	chk := checker.New(10 * time.Second)
	sched := scheduler.New(cfg.Targets, chk, st, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go sched.Run(ctx)

	return serve(ctx, addr, st, logger)
}

func serve(ctx context.Context, addr string, st store.Store, logger *slog.Logger) error {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: api.New(st),
		// Without this, a client that trickles request headers in slowly
		// can tie up a connection indefinitely (Slowloris).
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("healthwatch listening", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logger.Info("shutting down")
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
