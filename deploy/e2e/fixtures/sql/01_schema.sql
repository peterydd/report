-- E2E seed data for the report application.
-- E2E 种子数据：两张表，外加几条固定记录，便于断言。
--
-- The MySQL container in docker-compose.e2e.yml mounts this directory at
-- /docker-entrypoint-initdb.d and executes every *.sql file in alphabetical
-- order on first boot. The file is idempotent (CREATE TABLE IF NOT EXISTS,
-- INSERT IGNORE) so re-running against a partially-initialised volume is safe.

CREATE TABLE IF NOT EXISTS e2e_orders (
    id        INT          NOT NULL PRIMARY KEY,
    customer  VARCHAR(64)  NOT NULL,
    amount    DECIMAL(10,2) NOT NULL,
    created   DATETIME     NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS e2e_summary (
    id        INT          NOT NULL PRIMARY KEY,
    total     INT          NOT NULL,
    label     VARCHAR(32)  NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO e2e_orders (id, customer, amount, created) VALUES
    (1, 'alice', 100.50, '2026-01-01 10:00:00'),
    (2, 'bob',   200.75, '2026-01-02 11:00:00'),
    (3, 'carol', 300.00, '2026-01-03 12:00:00');

INSERT IGNORE INTO e2e_summary (id, total, label) VALUES
    (1, 100, 'jan'),
    (2, 200, 'feb'),
    (3, 300, 'mar');
