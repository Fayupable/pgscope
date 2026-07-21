-- pgscope demo dataset: a realistic-looking SaaS/e-commerce schema with
-- deliberately embedded index/query issues, purely so every Insights tab
-- has something real to show. Run once against a disposable demo
-- database, never against a real production database.

BEGIN;

-- ── Schema ──────────────────────────────────────────────────────────

CREATE TABLE users (
                       id serial PRIMARY KEY,
                       email text NOT NULL,
                       name text NOT NULL,
                       status text NOT NULL DEFAULT 'active',
                       created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE products (
                          id serial PRIMARY KEY,
                          sku text NOT NULL,
                          name text NOT NULL,
                          price numeric(10,2) NOT NULL,
                          stock integer NOT NULL DEFAULT 0
);

CREATE TABLE orders (
                        id serial PRIMARY KEY,
                        user_id integer NOT NULL,
                        status text NOT NULL,
                        total_amount numeric(10,2) NOT NULL,
                        created_at timestamptz NOT NULL DEFAULT now()
);
-- No index on user_id here, on purpose: this is the Index Candidate
-- scenario, lots of lookups by user_id with only a sequential scan
-- available.

CREATE TABLE order_items (
                             id serial PRIMARY KEY,
                             order_id integer NOT NULL,
                             product_id integer NOT NULL,
                             quantity integer NOT NULL,
                             price numeric(10,2) NOT NULL
);
CREATE INDEX idx_order_items_order_id ON order_items (order_id);

CREATE TABLE payments (
                          id serial PRIMARY KEY,
                          order_id integer NOT NULL,
                          amount numeric(10,2) NOT NULL,
                          status text NOT NULL,
                          created_at timestamptz NOT NULL DEFAULT now()
);
-- Duplicate Index scenario: two indexes on the same column.
CREATE INDEX idx_payments_order_id ON payments (order_id);
CREATE INDEX idx_payments_order_id_dup ON payments (order_id);

CREATE TABLE audit_logs (
                            id serial PRIMARY KEY,
                            actor_id integer,
                            action text NOT NULL,
                            target_table text NOT NULL,
                            target_id integer,
                            created_at timestamptz NOT NULL DEFAULT now()
);
-- Unused Index scenario: nobody ever queries by target_table in this
-- demo, so this index will sit at zero scans.
CREATE INDEX idx_audit_logs_target_table ON audit_logs (target_table);

CREATE TABLE sessions (
                          id serial PRIMARY KEY,
                          user_id integer NOT NULL,
                          token text NOT NULL,
                          expires_at timestamptz NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);

CREATE TABLE reviews (
                         id serial PRIMARY KEY,
                         product_id integer NOT NULL,
                         user_id integer NOT NULL,
                         rating integer NOT NULL,
                         comment text,
                         created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_reviews_product_id ON reviews (product_id);

CREATE TABLE notifications (
                               id serial PRIMARY KEY,
                               user_id integer NOT NULL,
                               type text NOT NULL,
                               payload text,
                               read_at timestamptz,
                               created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user_id ON notifications (user_id);
-- Deep-offset pagination candidate: the companion usage-simulation
-- script will run repeated "ORDER BY created_at OFFSET N LIMIT 20"
-- queries against this table.

-- ── Bulk seed data ──────────────────────────────────────────────────

INSERT INTO users (email, name, status, created_at)
SELECT
    'user' || i || '@example.com',
    'Demo User ' || i,
    (ARRAY['active', 'active', 'active', 'suspended'])[floor(random() * 4 + 1)],
    now() - (random() * interval '365 days')
FROM generate_series(1, 4000) i;

INSERT INTO products (sku, name, price, stock)
SELECT
    'SKU-' || lpad(i::text, 6, '0'),
    'Demo Product ' || i,
    (random() * 200 + 5)::numeric(10,2),
    floor(random() * 500)
FROM generate_series(1, 1000) i;

INSERT INTO orders (user_id, status, total_amount, created_at)
SELECT
    floor(random() * 4000 + 1),
    (ARRAY['pending', 'processing', 'shipped', 'delivered', 'cancelled'])[floor(random() * 5 + 1)],
    (random() * 500 + 10)::numeric(10,2),
    now() - (random() * interval '180 days')
FROM generate_series(1, 25000) i;

INSERT INTO order_items (order_id, product_id, quantity, price)
SELECT
    floor(random() * 25000 + 1),
    floor(random() * 1000 + 1),
    floor(random() * 5 + 1),
    (random() * 100 + 5)::numeric(10,2)
FROM generate_series(1, 55000) i;

INSERT INTO payments (order_id, amount, status, created_at)
SELECT
    floor(random() * 25000 + 1),
    (random() * 500 + 10)::numeric(10,2),
    (ARRAY['pending', 'completed', 'failed', 'refunded'])[floor(random() * 4 + 1)],
    now() - (random() * interval '180 days')
FROM generate_series(1, 25000) i;

INSERT INTO audit_logs (actor_id, action, target_table, target_id, created_at)
SELECT
    floor(random() * 4000 + 1),
    (ARRAY['create', 'update', 'delete', 'login'])[floor(random() * 4 + 1)],
    (ARRAY['orders', 'payments', 'users', 'products'])[floor(random() * 4 + 1)],
    floor(random() * 25000 + 1),
    now() - (random() * interval '365 days')
FROM generate_series(1, 30000) i;

INSERT INTO sessions (user_id, token, expires_at)
SELECT
    floor(random() * 4000 + 1),
    md5(random()::text || i::text),
    now() + (random() * interval '30 days')
FROM generate_series(1, 4000) i;

INSERT INTO reviews (product_id, user_id, rating, comment, created_at)
SELECT
    floor(random() * 1000 + 1),
    floor(random() * 4000 + 1),
    floor(random() * 5 + 1),
    'Demo review text for testing purposes, entry ' || i,
    now() - (random() * interval '300 days')
FROM generate_series(1, 12000) i;

INSERT INTO notifications (user_id, type, payload, read_at, created_at)
SELECT
    floor(random() * 4000 + 1),
    (ARRAY['order_update', 'payment_received', 'promo', 'system'])[floor(random() * 4 + 1)],
    '{"demo": true, "seq": ' || i || '}',
    CASE WHEN random() > 0.5 THEN now() - (random() * interval '10 days') ELSE NULL END,
    now() - (random() * interval '90 days')
FROM generate_series(1, 44000) i;

-- ── Slow trigger scenario ───────────────────────────────────────────
-- Attached AFTER seeding on purpose: seeding 25,000 orders through a
-- trigger that sleeps 10ms each would take minutes for no reason. This
-- only fires for orders inserted from now on, e.g. by the companion
-- usage-simulation script.

CREATE OR REPLACE FUNCTION log_order_audit() RETURNS trigger AS $$
BEGIN
    PERFORM pg_sleep(0.01); -- simulates non-trivial audit-logging work
INSERT INTO audit_logs (actor_id, action, target_table, target_id, created_at)
VALUES (NEW.user_id, 'create', 'orders', NEW.id, now());
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_log_order_audit
    AFTER INSERT ON orders
    FOR EACH ROW
    EXECUTE FUNCTION log_order_audit();

COMMIT;

ANALYZE users;
ANALYZE products;
ANALYZE orders;
ANALYZE order_items;
ANALYZE payments;
ANALYZE audit_logs;
ANALYZE sessions;
ANALYZE reviews;
ANALYZE notifications;