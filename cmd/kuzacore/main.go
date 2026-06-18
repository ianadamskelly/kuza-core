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

	"kuza-core/internal/config"
	"kuza-core/internal/database"
	"kuza-core/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.Load()

	var db *database.DB
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		connected, err := database.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("connect database", "error", err)
			os.Exit(1)
		}
		db = connected
		defer db.Close()

		if err := db.Migrate(ctx); err != nil {
			logger.Error("run migrations", "error", err)
			os.Exit(1)
		}

		if err := db.BootstrapOwner(ctx, cfg.Bootstrap); err != nil {
			logger.Error("bootstrap owner", "error", err)
			os.Exit(1)
		}
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewServer(cfg, logger, db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("kuza core api listening", "addr", cfg.Addr, "env", cfg.Env)
		errs <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
