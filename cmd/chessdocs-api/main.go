// Command chessdocs-api serves the contribution endpoint behind
// api.chessdocs.org: it hands readers the markdown source of a docs page and
// turns their edits into pull requests against the chessdocs repository.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dragunovartem99/chessdocs-api/internal/api"
	"github.com/dragunovartem99/chessdocs-api/internal/config"
	"github.com/dragunovartem99/chessdocs-api/internal/github"
)

// shutdownGrace is how long in-flight requests have to finish once a signal
// arrives — comfortably more than the GitHub client's own timeout.
const shutdownGrace = 20 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: api.NewServer(api.Options{
			GitHub:        github.New(cfg.GitHub),
			AllowedOrigin: cfg.AllowedOrigin,
			Logger:        log,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	// Stop serving on the first SIGINT or SIGTERM; a second one is left to the
	// default handler so an impatient operator can still kill the process.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listening := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", cfg.Addr))
		listening <- server.ListenAndServe()
	}()

	select {
	case err := <-listening:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		stop()
		log.Info("shutting down", slog.Duration("grace", shutdownGrace))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
