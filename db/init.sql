-- Single source of truth for the database schema.
-- Runs automatically on first startup of the TimescaleDB container
-- (mounted into /docker-entrypoint-initdb.d/; only executes on a fresh data dir).

CREATE TABLE IF NOT EXISTS ohlcv_1m (
    symbol        TEXT             NOT NULL,
    window_start  TIMESTAMPTZ      NOT NULL,
    open          DOUBLE PRECISION NOT NULL,
    high          DOUBLE PRECISION NOT NULL,
    low           DOUBLE PRECISION NOT NULL,
    close         DOUBLE PRECISION NOT NULL,
    volume        DOUBLE PRECISION NOT NULL,
    maker_volume  DOUBLE PRECISION NOT NULL,
    taker_volume  DOUBLE PRECISION NOT NULL,
    vwap          DOUBLE PRECISION NOT NULL,
    trade_count   BIGINT           NOT NULL,
    PRIMARY KEY (symbol, window_start)
);

SELECT create_hypertable('ohlcv_1m', 'window_start', if_not_exists => TRUE);
