package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/api"
	"github.com/kay-kewl/trendstream/internal/auth"
	kafkabroker "github.com/kay-kewl/trendstream/internal/broker/kafka"
	"github.com/kay-kewl/trendstream/internal/config"
	"github.com/kay-kewl/trendstream/internal/httpserver"
	"github.com/kay-kewl/trendstream/internal/ingest"
	"github.com/kay-kewl/trendstream/internal/logging"
	"github.com/kay-kewl/trendstream/internal/metrics"
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

	appMetrics := metrics.New()

	aggregatorConfig := aggregator.DefaultConfig()
	aggregatorConfig.ShardCount = cfg.ShardCount
	aggregatorConfig.Window.MaxUniqueQueries = cfg.MaxUniqueQueries
	aggregatorConfig.Window.MaxUniqueQueriesPerBucket = cfg.MaxUniqueQueriesPerBucket
	aggregatorConfig.Window.PerActorQueryLimit = cfg.PerActorQueryLimit

	trendAggregator, err := aggregator.New(aggregatorConfig)
	if err != nil {
		return err
	}

	stopListStore := stoplist.NewFileStore(cfg.StopListPath)

	stopListService, err := stoplist.NewService(stopListStore)
	if err != nil {
		return err
	}

	appMetrics.SetStopListRules(len(stopListService.Terms()))

	eventProcessor := ingest.NewProcessorWithObserver(trendAggregator, stopListService, appMetrics)
	httpEventProcessor := ingest.NewHTTPProcessor(eventProcessor)

	initialSnapshot := snapshot.Empty(startedAt)
	snapshotPublisher := snapshot.NewPublisher(initialSnapshot)

	go refreshSnapshots(rootCtx, logger, trendAggregator, stopListService, snapshotPublisher, appMetrics)

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

	adminEventsHandler := api.NewAdminEventsHandler(httpEventProcessor, adminAuth)
	adminEventsHandler.Register(adminMux)

	adminMux.Handle("GET /metrics", appMetrics.Handler())
	if cfg.PPROFEnabled {
		registerPPROF(adminMux)
	}

	publicHandler := api.MetricsMiddleware("public", appMetrics)(publicMux)
	adminHandler := api.MetricsMiddleware("admin", appMetrics)(adminMux)

	publicServer := httpserver.New(
		httpserver.ServerConfig{
			Addr: cfg.HTTPAddr,
			Name: "public",
		},
		publicHandler,
		logger,
	)

	adminServer := httpserver.New(
		httpserver.ServerConfig{
			Addr: cfg.AdminAddr,
			Name: "admin",
		},
		adminHandler,
		logger,
	)

	var kafkaConsumer *kafkabroker.Consumer
	if cfg.KafkaEnabled {
		kafkaConsumer, err = kafkabroker.NewConsumer(
			kafkabroker.ConsumerConfig{
				Brokers:  cfg.KafkaBrokers,
				Topic:    cfg.KafkaTopic,
				GroupID:  cfg.KafkaGroupID,
				ClientID: cfg.KafkaClientID,
			},
			eventProcessor,
			logger,
			appMetrics,
		)
		if err != nil {
			return err
		}

		defer kafkaConsumer.Close()
	}

	errCh := make(chan error, 3)

	go serveHTTP(errCh, logger, "public", publicServer)
	go serveHTTP(errCh, logger, "admin", adminServer)

	if kafkaConsumer != nil {
		go serveKafka(rootCtx, errCh, logger, kafkaConsumer)
	}

	logger.Info(
		"service started",
		slog.String("service", cfg.ServiceName),
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("admin_addr", cfg.AdminAddr),
		slog.String("log_level", cfg.LogLevel),
		slog.String("stoplist_path", cfg.StopListPath),
		slog.Bool("kafka_enabled", cfg.KafkaEnabled),
		slog.String("kafka_topic", cfg.KafkaTopic),
		slog.String("kafka_group_id", cfg.KafkaGroupID),
		slog.Int("shard_count", cfg.ShardCount),
		slog.Int("max_unique_queries", cfg.MaxUniqueQueries),
		slog.Int("max_unique_queries_per_bucket", cfg.MaxUniqueQueriesPerBucket),
		slog.Int64("per_actor_query_limit", cfg.PerActorQueryLimit),
		slog.Bool("pprof_enabled", cfg.PPROFEnabled),
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
	stopListService *stoplist.Service,
	publisher *snapshot.Publisher,
	appMetrics *metrics.Metrics,
) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	rebuild := func(now time.Time) {
		startedAt := time.Now()
		items := trendAggregator.TopFilteredAt(snapshot.MaxLimit, now, func(item aggregator.Item) bool {
			return !stopListService.Contains(item.Query)
		})

		next, err := snapshot.New(items, now, snapshot.DefaultOptions())
		if err != nil {
			logger.Error("failed to build trends snapshot", slog.Any("error", err))
			return
		}

		publisher.Publish(next)

		appMetrics.ObserveSnapshotRebuild(time.Since(startedAt), next.GeneratedAt, len(next.Items))
		appMetrics.SetAggregatorStats(
			trendAggregator.UniqueQueriesAt(now),
			trendAggregator.WindowEventsAt(now),
			trendAggregator.ActorCountersAt(now),
		)
		appMetrics.SetStopListRules(len(stopListService.Terms()))
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

func filterStopListedItems(items []aggregator.Item, stopListService *stoplist.Service) []aggregator.Item {
	if len(items) == 0 || stopListService == nil {
		return items
	}

	filtered := items[:0]
	for _, item := range items {
		if stopListService.Contains(item.Query) {
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered
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

func serveKafka(
	ctx context.Context,
	errCh chan<- error,
	logger *slog.Logger,
	consumer *kafkabroker.Consumer,
) {
	if err := consumer.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}

		errCh <- err
		return
	}

	logger.Info("kafka consumer stopped")
}

func registerPPROF(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.Handle("GET /debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("GET /debug/pprof/block", pprof.Handler("block"))
	mux.Handle("GET /debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("GET /debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate"))
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
