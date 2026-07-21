-- Runs real queries against the demo dataset so pg_stat_statements and
-- pg_stat_user_tables accumulate genuine call history, every Insights
-- tab should show real signal after this runs. Safe only against the
-- demo database, never production.

DO $$
DECLARE
i integer;
BEGIN
    -- Index Candidate signal: orders.user_id has no index, this builds
    -- up seq_scan history against it.
FOR i IN 1..200 LOOP
        PERFORM * FROM orders WHERE user_id = floor(random() * 4000 + 1);
END LOOP;

    -- A handful of primary-key lookups too, so orders also has
    -- idx_scan > 0 (pgscope requires some existing index usage before
    -- it'll suggest a new one — a table with zero index usage at all
    -- reads as "not enough signal yet", not as a candidate).
FOR i IN 1..80 LOOP
        PERFORM * FROM orders WHERE id = floor(random() * 25000 + 1);
END LOOP;

    -- Pagination warning signal: OFFSET-based paging through
    -- notifications with widely varying offsets, producing the
    -- execution-time variance pgscope looks for.
FOR i IN 1..150 LOOP
        PERFORM * FROM notifications ORDER BY created_at OFFSET floor(random() * 40000) LIMIT 20;
END LOOP;

    -- A few more realistic lookups so Top Queries has variety.
FOR i IN 1..100 LOOP
        PERFORM * FROM users WHERE id = floor(random() * 4000 + 1);
END LOOP;

FOR i IN 1..100 LOOP
        PERFORM * FROM reviews WHERE product_id = floor(random() * 1000 + 1);
END LOOP;
END $$;

-- Function/Trigger cost signal: each insert here fires
-- trg_log_order_audit, which sleeps ~10ms and writes to audit_logs.
-- 150 calls mirrors the exact scenario already validated earlier
-- (150 calls, ~11ms self-time average).
DO $$
DECLARE
i integer;
BEGIN
FOR i IN 1..150 LOOP
        INSERT INTO orders (user_id, status, total_amount, created_at)
        VALUES (floor(random() * 4000 + 1), 'pending', (random() * 200 + 10)::numeric(10,2), now());
END LOOP;
END $$;

ANALYZE orders;
ANALYZE notifications;
ANALYZE users;
ANALYZE reviews;