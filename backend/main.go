// Command backend is the Healthwatch API: a small CRUD service backed by
// PostgreSQL, serving its own Vue frontend build in production.
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

	"healthwatch/backend/db"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(*addr, logger); err != nil {
		logger.Error("backend exited with an error", "error", err)
		os.Exit(1)
	}
}

func run(addr string, logger *slog.Logger) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errRequiredEnv("DATABASE_URL")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.NewPGStore(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	httpServer := &http.Server{
		Addr:    addr,
		Handler: NewServer(store, frontendFS()),
		// Without this, a client that trickles request headers in slowly
		// can tie up a connection indefinitely (Slowloris).
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("backend listening", "addr", addr)
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

type errRequiredEnv string

func (e errRequiredEnv) Error() string {
	return "required environment variable " + string(e) + " is not set"
}
