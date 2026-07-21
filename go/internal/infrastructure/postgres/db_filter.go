package postgres

// currentDbFilter restricts a pg_stat_statements query to the database
// this connection is actually pointed at — shared by every collector
// that reads from pg_stat_statements, so the fragment lives in one
// place instead of being copy-pasted per query.
const currentDbFilter = `dbid = (SELECT oid FROM pg_database WHERE datname = current_database())`
