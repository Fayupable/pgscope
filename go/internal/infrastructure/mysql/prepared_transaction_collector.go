package mysql

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/fayupable/pgscope/internal/domain"
)

// preparedTransactionQuery reads XA transactions still in the PREPARED
// state from information_schema.innodb_trx — MySQL's equivalent of
// pg_prepared_xacts. trx_id is used as the transaction identifier (a
// proper XA GID would require the XA RECOVER statement, which isn't a
// plain SELECT and so is out of scope for a read-only tool). Owner and
// database come from a left join to performance_schema.threads, since
// innodb_trx alone doesn't carry the connecting user/database.
const preparedTransactionQuery = `
SELECT
    t.trx_id,
    COALESCE(th.PROCESSLIST_DB, ''),
    COALESCE(th.PROCESSLIST_USER, ''),
    TIMESTAMPDIFF(SECOND, t.trx_started, NOW())
FROM information_schema.innodb_trx t
LEFT JOIN performance_schema.threads th ON th.PROCESSLIST_ID = t.trx_mysql_thread_id
WHERE t.trx_state = 'PREPARED'
`

// PreparedTransactionCollector reads orphaned XA (two-phase-commit)
// transactions from information_schema.innodb_trx. It has no opinion on
// how long is "too long" — that judgment belongs to
// domain.DetectPreparedTransactionWarnings, unchanged from the Postgres
// adapter.
type PreparedTransactionCollector struct {
	db *sql.DB
}

func NewPreparedTransactionCollector(db *sql.DB) *PreparedTransactionCollector {
	return &PreparedTransactionCollector{db: db}
}

func (c *PreparedTransactionCollector) Fetch(ctx context.Context) ([]domain.PreparedTransactionInfo, error) {
	rows, err := c.db.QueryContext(ctx, preparedTransactionQuery)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	transactions := make([]domain.PreparedTransactionInfo, 0)
	for rows.Next() {
		var trxID uint64
		var tx domain.PreparedTransactionInfo
		if err := rows.Scan(&trxID, &tx.Database, &tx.Owner, &tx.AgeSeconds); err != nil {
			return nil, err
		}
		tx.GID = strconv.FormatUint(trxID, 10)
		transactions = append(transactions, tx)
	}

	return transactions, rows.Err()
}
