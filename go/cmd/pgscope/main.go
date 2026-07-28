package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fayupable/pgscope/internal/application/service"
	"github.com/fayupable/pgscope/internal/infrastructure/config"
	"github.com/fayupable/pgscope/internal/infrastructure/history"
	"github.com/fayupable/pgscope/internal/infrastructure/postgres"
	"github.com/fayupable/pgscope/internal/infrastructure/sse"
	presentationhttp "github.com/fayupable/pgscope/internal/presentation/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("pgscope exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{ConnString: cfg.DatabaseURL})
	if err != nil {
		return err
	}
	defer pool.Close()

	historyStore, err := history.NewSQLiteStore(cfg.HistoryDBPath)
	if err != nil {
		return fmt.Errorf("open history store: %w", err)
	}
	defer func() { _ = historyStore.Close() }()

	broadcaster := sse.NewBroadcaster()
	poller := buildPoller(pool, broadcaster, historyStore, cfg)
	insightsService := buildInsightsService(pool)
	server := buildServer(broadcaster, poller, insightsService, cfg)

	go poller.Run(ctx)
	go runHistoryPruner(ctx, historyStore, cfg.HistoryRetention, cfg.HistoryMaxDBSizeBytes)

	return runServer(ctx, server)
}

func buildPoller(pool *pgxpool.Pool, broadcaster *sse.Broadcaster, historyStore *history.SQLiteStore, cfg config.Config) *service.Poller {
	collector := postgres.NewCollector(pool)
	monitoringService := service.NewMonitoringService(collector)

	dbStatsCollector := postgres.NewDatabaseStatsCollector(pool)
	publisher := sse.NewSessionPublisher(broadcaster)

	return service.NewPoller(
		monitoringService,
		dbStatsCollector,
		publisher,
		historyStore,
		cfg.PollInterval,
		cfg.HistoryRecordInterval,
		cfg.HistoryMaxSessionsPerSnapshot,
	)
}

func buildInsightsService(pool *pgxpool.Pool) *service.InsightsService {
	insightsCollector := postgres.NewInsightsCollector(pool)
	return service.NewInsightsService(insightsCollector)
}

func buildServer(broadcaster *sse.Broadcaster, poller *service.Poller, insightsService *service.InsightsService, cfg config.Config) *http.Server {
	mux := presentationhttp.NewRouter(broadcaster, poller, insightsService, cfg)

	return &http.Server{
		Addr:              net.JoinHostPort("", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func runServer(ctx context.Context, server *http.Server) error {
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

const historyPruneInterval = 6 * time.Hour

// runHistoryPruner periodically deletes periodic snapshots older than the
// configured retention window, in small batches (see PruneOlderThan) so it
// never causes a noticeable CPU or lock spike. It also caps the number of
// incident snapshots kept (see PruneExcessIncidents), since those are never
// pruned by age, and enforces a hard disk-size backstop (EnforceMaxSize) on
// top of the age-based retention, in case an unusually busy monitored
// database outpaces it.
func runHistoryPruner(ctx context.Context, store *history.SQLiteStore, retention time.Duration, maxDBSizeBytes int64) {
	ticker := time.NewTicker(historyPruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-retention)
			if err := store.PruneOlderThan(ctx, cutoff); err != nil {
				slog.Error("history prune failed", "error", err)
			}
			if err := store.PruneExcessIncidents(ctx, history.MaxIncidentRows); err != nil {
				slog.Error("incident prune failed", "error", err)
			}
			if err := store.EnforceMaxSize(ctx, maxDBSizeBytes); err != nil {
				slog.Error("history size enforcement failed", "error", err)
			}
		}
	}
}
