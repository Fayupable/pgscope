package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxSessions = 250
const runDuration = 60 * time.Second

// Row ranges within test_orders (seeded with ids 1..300):
//
//	1..5    -> "hot" rows: targeted by many workers, forms hub/fan-in nodes
//	6..45   -> "chain" rows: each worker locks its own + the next worker's
//	           row, forming multi-hop waiting chains
//	46..295 -> "independent" rows: one per worker, untouched by anyone else
const (
	hotRowCount          = 5
	chainRowCount        = 40
	rowIDOffset          = 1
	hotRowsStart         = rowIDOffset
	chainRowsStart       = hotRowsStart + hotRowCount
	independentRowsStart = chainRowsStart + chainRowCount
	independentRowCount  = maxSessions
)

func main() {
	if err := run(); err != nil {
		slog.Error("loadgen exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	connString := os.Getenv("LOADGEN_DATABASE_URL")
	if connString == "" {
		return fmt.Errorf("LOADGEN_DATABASE_URL is required (must be a write-capable/superuser connection, not pgscope_agent)")
	}

	httpPort := os.Getenv("LOADGEN_HTTP_PORT")
	if httpPort == "" {
		httpPort = "9091"
	}

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}
	poolConfig.MaxConns = int32(maxSessions) + 10

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /run", handleRun(pool))

	slog.Info("loadgen HTTP trigger listening", "port", httpPort, "endpoint", "POST /run?sessions=N", "run_duration", runDuration)
	return http.ListenAndServe(":"+httpPort, mux)
}

func handleRun(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := parseSessionsParam(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Each /run call gets its own timeout, independent of the HTTP
		// request lifecycle — workers keep running for runDuration even
		// after this handler returns the response.
		runCtx, cancel := context.WithTimeout(context.Background(), runDuration)

		for i := 0; i < sessions; i++ {
			go runWorker(runCtx, pool, i)
		}
		go func() {
			<-runCtx.Done()
			cancel()
			slog.Info("load run finished", "sessions", sessions)
		}()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"spawned":          sessions,
			"duration_seconds": int(runDuration.Seconds()),
		}); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
	}
}

func parseSessionsParam(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("sessions")
	if raw == "" {
		return 0, fmt.Errorf("missing required query param: sessions")
	}

	sessions, err := strconv.Atoi(raw)
	if err != nil || sessions <= 0 {
		return 0, fmt.Errorf("sessions must be a positive integer")
	}
	if sessions > maxSessions {
		return 0, fmt.Errorf("sessions cannot exceed %d", maxSessions)
	}

	return sessions, nil
}

func runWorker(ctx context.Context, pool *pgxpool.Pool, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			runOneOperation(ctx, pool, workerID)
			select {
			case <-ctx.Done():
				return
			case <-time.After(randomDelay(500*time.Millisecond, 2*time.Second)):
			}
		}
	}
}

// runOneOperation assigns each worker one of three behaviors based on its
// ID, producing a realistic mixed topology in a single run:
//   - hub workers (2/5) all compete for a small pool of "hot" rows, forming
//     fan-in nodes that many sessions wait on at once
//   - chain workers (2/5) lock their own row plus the next worker's row,
//     forming genuine multi-hop waiting chains (A waits on B waits on C)
//   - independent workers (1/5) touch a row nobody else uses, appearing as
//     isolated nodes with no relationships
func runOneOperation(ctx context.Context, pool *pgxpool.Pool, workerID int) {
	holdDuration := randomDelay(3*time.Second, 9*time.Second)

	tx, err := pool.Begin(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("worker failed to begin tx", "worker", workerID, "error", err)
		}
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	switch workerID % 5 {
	case 0, 1:
		runHubOperation(ctx, tx, workerID)
	case 2, 3:
		runChainOperation(ctx, tx, workerID)
	default:
		runIndependentOperation(ctx, tx, workerID)
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(holdDuration):
	}

	if err := tx.Commit(context.Background()); err != nil {
		slog.Warn("worker failed to commit", "worker", workerID, "error", err)
	}
}

func runHubOperation(ctx context.Context, tx pgx.Tx, workerID int) {
	targetRow := hotRowsStart + ((workerID / 5) % hotRowCount)
	updateRow(ctx, tx, workerID, targetRow)
}

func runChainOperation(ctx context.Context, tx pgx.Tx, workerID int) {
	homeRow := chainRowsStart + (workerID % chainRowCount)
	nextRow := chainRowsStart + ((workerID + 1) % chainRowCount)
	updateRow(ctx, tx, workerID, homeRow)
	updateRow(ctx, tx, workerID, nextRow)
}

func runIndependentOperation(ctx context.Context, tx pgx.Tx, workerID int) {
	row := independentRowsStart + (workerID % independentRowCount)
	updateRow(ctx, tx, workerID, row)
}

func updateRow(ctx context.Context, tx pgx.Tx, workerID int, rowID int) {
	if _, err := tx.Exec(ctx, "UPDATE test_orders SET status = $1 WHERE id = $2", randomStatus(), rowID); err != nil {
		if ctx.Err() == nil {
			slog.Warn("worker failed to update row", "worker", workerID, "row", rowID, "error", err)
		}
	}
}

func randomStatus() string {
	statuses := []string{"pending", "processing", "done", "cancelled"}
	return statuses[rand.Intn(len(statuses))]
}

func randomDelay(min, max time.Duration) time.Duration {
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
