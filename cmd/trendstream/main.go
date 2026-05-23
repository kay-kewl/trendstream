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

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/api"
	"github.com/kay-kewl/trendstream/internal/auth"
	"github.com/kay-kewl/trendstream/internal/config"
	"github.com/kay-kewl/trendstream/internal/httpserver"
	"github.com/kay-kewl/trendstream/internal/logging"
	"github.com/kay-kewl/trendstream/internal/snapshot"
	"github.com/kay-kewl/trendstream/internal/stoplist"
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

	rootCtx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	trendAggregator, err := aggregator.New(aggregator.DefaultConfig())
	if err != nil {
		return err
	}

	stopListStore := stoplist.NewFileStore(cfg.StopListPath)

	stopListService, err := stoplist.NewService(stopListStore)
	if err != nil {
		return err
	}

	initialSnapshot := snapshot.Empty(startedAt)
	snapshotPublisher := snapshot.NewPublisher(initialSnapshot)

	go refreshSnapshots(rootCtx, logger, trendAggregator, snapshotPublisher)

	publicMux := http.NewServeMux()
	adminMux := http.NewServeMux()

	healthHandler := api.NewHealthHandler(cfg.ServiceName, startedAt)
	healthHandler.Register(publicMux)
	healthHandler.Register(adminMux)

	trendsHandler := api.NewTrendsHandler(snapshotPublisher)
	trendsHandler.Register(publicMux)

	adminAuth := auth.NewTokenAuth(cfg.AdminToken)

	adminStopListHandler := api.NewAdminStopListHandler(stopListService, adminAuth)
	adminStopListHandler.Register(adminMux)

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
		slog.String("stoplist_path", cfg.StopListPath),
	)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		cancelRoot()
		return err
	}

	cancelRoot()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := shutdownServers(shutdownCtx, logger, publicServer, adminServer); err != nil {
		return err
	}

	logger.Info("service stopped gracefully")
	return nil
}

func refreshSnapshots(
	ctx context.Context,
	logger *slog.Logger,
	trendAggregator *aggregator.Aggregator,
	publisher *snapshot.Publisher,
) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	rebuild := func(now time.Time) {
		items := trendAggregator.TopAt(snapshot.MaxLimit, now)

		next, err := snapshot.New(items, now, snapshot.DefaultOptions())
		if err != nil {
			logger.Error("failed to build trends snapshot", slog.Any("error", err))
			return
		}

		publisher.Publish(next)
	}

	rebuild(time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			logger.Info("snapshot refresher stopped")
			return
		case now := <-ticker.C:
			rebuild(now.UTC())
		}
	}
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
