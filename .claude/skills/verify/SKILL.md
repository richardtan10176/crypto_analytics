---
name: verify
description: Run the crypto_analytics pipeline end-to-end and confirm candles land in TimescaleDB.
---

# Verify the pipeline end-to-end

Surface: two host-run binaries against dockerized Kafka + TimescaleDB, observed via psql.

```sh
open -a Docker                     # if the daemon isn't up (macOS)
make docker-up                     # kafka + timescaledb; init.sql runs only on a fresh data dir
make build
./bin/ingester  > /tmp/ingester.log 2>&1 &
./bin/consumer  > /tmp/consumer.log 2>&1 &
```

Wait ~90s (a minute boundary must pass before the first window flushes), then:

```sh
docker compose exec -T timescaledb psql -U user -d trades \
  -c "SELECT symbol, window_start, open, high, low, close, volume, trade_count FROM ohlcv_1m ORDER BY window_start;"
```

Sanity: low ≤ open,close ≤ high; volume = maker_volume + taker_volume; vwap in [low, high]; one row per (symbol, minute).

Useful probes:
- Consumer lag / liveness: `docker compose exec -T kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group analytics`
- Graceful shutdown: `pkill -TERM -f bin/consumer` → log line "consumer stopped" + a partial candle row for the in-flight minute (final flush).
- Restart mid-window: the partial row gets **replaced** when the window completes (upsert), but undercounts pre-restart trades — that's the documented known limitation (offsets commit per message, not at flush).

Gotchas:
- Both binaries load `.env`; consumer needs `TIMESCALE_DSN`.
- Ingest rate should be tens of msgs/sec. If it's ~1 msg/s, someone reintroduced the kafka-go Writer default `BatchTimeout` (1s) — the producer must set it low.
- `raw Kafka events` sample: `docker compose exec -T kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic trades --max-messages 2 --from-beginning --timeout-ms 15000` (protobuf, so mostly binary).
