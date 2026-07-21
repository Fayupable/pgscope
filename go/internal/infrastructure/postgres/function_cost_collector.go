package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const functionStatsLimit = 20

// functionStatsQuery reads per-function call statistics. This is only
// populated when track_functions is set to 'pl' or 'all' — if it's
// 'none' (the default), this simply returns no rows, which the
// orchestrator distinguishes via a separate settings check rather than
// guessing from an empty result.
const functionStatsQuery = `
SELECT
    f.funcid,
    p.proname AS function_name,
    f.calls,
    f.total_time,
    f.self_time
FROM pg_stat_user_functions f
JOIN pg_proc p ON p.oid = f.funcid
WHERE f.calls >= $1
ORDER BY (f.self_time / NULLIF(f.calls, 0)) DESC
LIMIT $2
`

// triggerFunctionsQuery maps function OIDs to the tables they're
// attached to as triggers, straight from the catalog — no text parsing.
// Internal triggers (tgisinternal, e.g. those backing foreign keys) and
// disabled triggers are excluded, since a disabled trigger never
// actually runs regardless of what its function's stats say.
const triggerFunctionsQuery = `
SELECT DISTINCT
    t.tgfoid,
    c.relname
FROM pg_trigger t
JOIN pg_class c ON c.oid = t.tgrelid
WHERE NOT t.tgisinternal
  AND t.tgenabled IN ('O', 'A')
`

const trackFunctionsSettingQuery = `SELECT setting FROM pg_settings WHERE name = 'track_functions'`

// FunctionCostCollector reads function/trigger call statistics from the
// system catalog. It has no opinion on whether a function is "too
// expensive" — that judgment belongs to domain.DetectExpensiveFunctions.
type FunctionCostCollector struct {
	pool *pgxpool.Pool
}

func NewFunctionCostCollector(pool *pgxpool.Pool) *FunctionCostCollector {
	return &FunctionCostCollector{pool: pool}
}

// TrackFunctionsSetting reports the current value of the track_functions
// GUC. pgscope never changes this — it only ever reads it, so the caller
// can tell the user to enable it themselves if it's "none".
func (c *FunctionCostCollector) TrackFunctionsSetting(ctx context.Context) (string, error) {
	var setting string
	err := c.pool.QueryRow(ctx, trackFunctionsSettingQuery).Scan(&setting)
	return setting, err
}

func (c *FunctionCostCollector) FetchFunctionStats(ctx context.Context) ([]domain.FunctionCallStats, error) {
	triggerTablesByFunc, err := c.fetchTriggerTables(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := c.pool.Query(ctx, functionStatsQuery, domain.MinCallsForFunctionCostWarning, functionStatsLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]domain.FunctionCallStats, 0)
	for rows.Next() {
		var funcid uint32
		var s domain.FunctionCallStats
		if err := rows.Scan(&funcid, &s.Function, &s.Calls, &s.TotalTimeMs, &s.SelfTimeMs); err != nil {
			return nil, err
		}

		if tables, ok := triggerTablesByFunc[funcid]; ok {
			s.IsTrigger = true
			s.TriggerTables = tables
		}

		stats = append(stats, s)
	}

	return stats, rows.Err()
}

func (c *FunctionCostCollector) fetchTriggerTables(ctx context.Context) (map[uint32][]string, error) {
	rows, err := c.pool.Query(ctx, triggerFunctionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint32][]string)
	for rows.Next() {
		var funcid uint32
		var table string
		if err := rows.Scan(&funcid, &table); err != nil {
			return nil, err
		}
		result[funcid] = append(result[funcid], table)
	}

	return result, rows.Err()
}
