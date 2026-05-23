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

	"github.com/kay-kewl/trendstream/internal/api"
	"github.com/kay-kewl/trendstream/internal/config"
	"github.com/kay-kewl/trendstream/internal/httpserver"
	"github.com/kay-kewl/trendstream/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("service stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	startedAt := time.Now().UTC()

	publicMux := http.NewServeMux()
	adminMux := http.NewServeMux()

	healthHandler := api.NewHealthHandler(cfg.ServiceName, startedAt)
	healthHandler.Register(publicMux)
	healthHandler.Register(adminMux)

	publicServer := httpserver.New(
		httpserver.ServerConfig{
			Addr: cfg.HTTPAddr,
			Name: "public",
		},
		publicMux,
		logger,
	)

	adminServer := httpserver.New(
		httpserver.ServerConfig{
			Addr: cfg.AdminAddr,
			Name: "admin",
		},
		adminMux,
		logger,
	)

	errCh := make(chan error, 2)

	go serveHTTP(errCh, logger, "public", publicServer)
	go serveHTTP(errCh, logger, "admin", adminServer)

	logger.Info(
		"service started",
		slog.String("service", cfg.ServiceName),
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("admin_addr", cfg.AdminAddr),
		slog.String("log_level", cfg.LogLevel),
	)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := shutdownServers(shutdownCtx, logger, publicServer, adminServer); err != nil {
		return err
	}

	logger.Info("service stopped gracefully")
	return nil
}

func serveHTTP(errCh chan<- error, logger *slog.Logger, name string, server *http.Server) {
	logger.Info(
		"HTTP server is listening",
		slog.String("server", name),
		slog.String("addr", server.Addr),
	)

	err := server.ListenAndServe()
	if err == nil {
		return
	}

	if errors.Is(err, http.ErrServerClosed) {
		logger.Info("HTTP server closed", slog.String("server", name))
		return
	}

	errCh <- err
}

func shutdownServers(ctx context.Context, logger *slog.Logger, servers ...*http.Server) error {
	var shutdownErr error

	for _, server := range servers {
		logger.Info("shutting down HTTP server", slog.String("addr", server.Addr))

		if err := server.Shutdown(ctx); err != nil {
			logger.Error(
				"failed to shutdown HTTP server",
				slog.String("addr", server.Addr),
				slog.Any("error", err),
			)

			shutdownErr = errors.Join(shutdownErr, err)
		}
	}

	return shutdownErr
}
