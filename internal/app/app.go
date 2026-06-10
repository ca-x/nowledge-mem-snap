package app

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/backup"
	"github.com/lib-x/nowledge-mem-snap/internal/config"
	"github.com/lib-x/nowledge-mem-snap/internal/history"
	"github.com/lib-x/nowledge-mem-snap/internal/scheduler"
	"github.com/lib-x/nowledge-mem-snap/internal/server"
	"github.com/lib-x/nowledge-mem-snap/version"
)

func Run(web embed.FS, args []string) int {
	logger := newLogger(dataDir())
	if len(args) > 1 && args[1] == "version" {
		fmt.Println(version.Full())
		return 0
	}
	if err := configureTimezone(logger); err != nil {
		logger.Error("timezone invalid", "error", err)
		return 1
	}
	if len(args) > 1 {
		switch args[1] {
		case "validate":
			_, _, err := load(logger)
			if err != nil {
				logger.Error("configuration invalid", "error", err)
				return 1
			}
			fmt.Println("configuration valid")
			return 0
		case "backup":
			tenant := ""
			task := config.DefaultTaskKey
			if len(args) > 2 {
				tenant = args[2]
			}
			if len(args) > 3 {
				task = args[3]
			}
			if tenant == "" {
				logger.Error("tenant is required: nowledge-mem-snap backup <tenant> [task]")
				return 1
			}
			_, store, err := load(logger)
			if err != nil {
				logger.Error("failed to load configuration", "error", err)
				return 1
			}
			defer store.Close()
			cfg, err := store.LoadUser(tenant)
			if err != nil {
				logger.Error("failed to load tenant configuration", "tenant", tenant, "error", err)
				return 1
			}
			historyStore := history.NewStoreWithRetention(store.Client(), config.TenantKey(tenant), cfg.HistoryLimit, cfg.HistoryRetentionDays)
			runner := backup.NewRunner(cfg, historyStore, logger)
			run, err := runner.RunTask(context.Background(), task)
			if err != nil {
				logger.Error("backup failed", "run", run.ID, "error", err)
				return 1
			}
			logger.Info("backup finished", "run", run.ID, "status", run.Status)
			return 0
		}
	}
	return serve(web, logger)
}

func serve(web embed.FS, logger *slog.Logger) int {
	cfg, store, err := load(logger)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		return 1
	}
	defer store.Close()
	if err := store.BootstrapFromEnv(); err != nil {
		logger.Error("failed to bootstrap admin user", "error", err)
		return 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schedulerManager := scheduler.NewManager(ctx, store, logger)
	schedulerManager.StartAll()
	defer schedulerManager.Stop()

	srv, err := server.New(ctx, cfg, store, web, logger, schedulerManager.Reload, schedulerManager.Snapshot)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		return 1
	}

	addr := fmt.Sprintf("%s:%d", cfg.Listen.Host, cfg.Listen.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("nowledge mem snap starting", "version", version.Full(), "addr", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			return 1
		}
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		return 1
	}
	return 0
}

func load(_ *slog.Logger) (config.Config, *config.Store, error) {
	store := config.NewStore(dataDir())
	cfg, err := store.Load()
	return cfg, store, err
}

func dataDir() string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(".", "data")
}
