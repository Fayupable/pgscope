package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fayupable/pgscope/internal/domain"
)

const preparedTransactionQuery = `
SELECT
    gid,
    database,
    owner,
    EXTRACT(EPOCH FROM now() - prepared)
FROM pg_prepared_xacts
`

// PreparedTransactionCollector reads two-phase-commit transactions still
// awaiting COMMIT PREPARED or ROLLBACK PREPARED, straight from
// pg_prepared_xacts — this view is not filtered by database, since a
// forgotten prepared transaction on any database in the cluster still
// holds locks and blocks vacuum cluster-wide. It has no opinion on how
// long is "too long" — that judgment belongs to
// domain.DetectPreparedTransactionWarnings.
type PreparedTransactionCollector struct {
	pool *pgxpool.Pool
}

func NewPreparedTransactionCollector(pool *pgxpool.Pool) *PreparedTransactionCollector {
	return &PreparedTransactionCollector{pool: pool}
}

func (c *PreparedTransactionCollector) Fetch(ctx context.Context) ([]domain.PreparedTransactionInfo, error) {
	rows, err := c.pool.Query(ctx, preparedTransactionQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]domain.PreparedTransactionInfo, 0)
	for rows.Next() {
		var tx domain.PreparedTransactionInfo
		if err := rows.Scan(&tx.GID, &tx.Database, &tx.Owner, &tx.AgeSeconds); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, rows.Err()
}
