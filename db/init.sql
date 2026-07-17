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

-- Hourly and daily rollups, computed automatically by TimescaleDB from
-- ohlcv_1m so the dashboard can render long ranges without pulling
-- thousands of 1-minute candles. materialized_only = false (real-time
-- aggregation) means querying these also reflects the still-filling current
-- bucket by unioning in unmaterialized raw rows, not just what the refresh
-- policy has already materialized — without it, this TimescaleDB version
-- defaults to materialized_only = true and a brand-new bucket stays empty
-- until the next scheduled refresh.
CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1h
WITH (timescaledb.continuous, timescaledb.materialized_only = false) AS
SELECT
    symbol,
    time_bucket('1 hour', window_start) AS window_start,
    first(open, window_start)  AS open,
    max(high)                  AS high,
    min(low)                   AS low,
    last(close, window_start)  AS close,
    sum(volume)                AS volume,
    sum(maker_volume)          AS maker_volume,
    sum(taker_volume)          AS taker_volume,
    CASE WHEN sum(volume) > 0 THEN sum(vwap * volume) / sum(volume) ELSE 0 END AS vwap,
    sum(trade_count)           AS trade_count
FROM ohlcv_1m
GROUP BY symbol, time_bucket('1 hour', window_start)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1h',
    start_offset => INTERVAL '3 days',
    end_offset   => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE);

CREATE MATERIALIZED VIEW IF NOT EXISTS ohlcv_1d
WITH (timescaledb.continuous, timescaledb.materialized_only = false) AS
SELECT
    symbol,
    time_bucket('1 day', window_start) AS window_start,
    first(open, window_start)  AS open,
    max(high)                  AS high,
    min(low)                   AS low,
    last(close, window_start)  AS close,
    sum(volume)                AS volume,
    sum(maker_volume)          AS maker_volume,
    sum(taker_volume)          AS taker_volume,
    CASE WHEN sum(volume) > 0 THEN sum(vwap * volume) / sum(volume) ELSE 0 END AS vwap,
    sum(trade_count)           AS trade_count
FROM ohlcv_1m
GROUP BY symbol, time_bucket('1 day', window_start)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('ohlcv_1d',
    start_offset => INTERVAL '90 days',
    end_offset   => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);
