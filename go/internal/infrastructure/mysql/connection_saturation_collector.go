package mysql

import (
	"context"
	"database/sql"

	"github.com/fayupable/pgscope/internal/domain"
)

// performance_schema.global_status / global_variables are used instead of
// `SHOW STATUS LIKE ?` / `SHOW VARIABLES LIKE ?` — the SHOW statement
// grammar doesn't accept a placeholder argument the way a normal SELECT
// does (the driver sends a literal "?" the server can't parse), while
// these are just regular queryable tables.
const statusQuery = `SELECT VARIABLE_VALUE FROM performance_schema.global_status WHERE VARIABLE_NAME = ?`
const variableQuery = `SELECT VARIABLE_VALUE FROM performance_schema.global_variables WHERE VARIABLE_NAME = ?`

// ConnectionSaturationCollector reads active connection count and the
// configured ceiling directly from MySQL's status/variable views — a
// purely descriptive fact with no judgment attached, same role as its
// Postgres counterpart. Judgment (when this is "too high") belongs to
// domain.NewConnectionSaturation, which is engine-agnostic and already
// shared with the Postgres adapter.
type ConnectionSaturationCollector struct {
	db *sql.DB
}

func NewConnectionSaturationCollector(db *sql.DB) *ConnectionSaturationCollector {
	return &ConnectionSaturationCollector{db: db}
}

func (c *ConnectionSaturationCollector) Fetch(ctx context.Context) (domain.ConnectionSaturation, error) {
	active, err := c.readInt(ctx, statusQuery, "Threads_connected")
	if err != nil {
		return domain.ConnectionSaturation{}, err
	}

	maxConns, err := c.readInt(ctx, variableQuery, "max_connections")
	if err != nil {
		return domain.ConnectionSaturation{}, err
	}

	return domain.NewConnectionSaturation(active, maxConns), nil
}

func (c *ConnectionSaturationCollector) readInt(ctx context.Context, query, name string) (int, error) {
	var value int
	if err := c.db.QueryRowContext(ctx, query, name).Scan(&value); err != nil {
		return 0, err
	}
	return value, nil
}
