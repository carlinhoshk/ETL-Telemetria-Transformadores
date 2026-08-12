#!/usr/bin/env bash
# E2E integration: MQTT -> ingestion (Go) -> PostgreSQL (bronze) -> dbt silver.
# Requires docker for the broker/postgres; uses the .venv for dbt.
# Asserts rows landed in bronze (raw_telemetry, measurements) and silver
# (stg_telemetry).
set -euo pipefail

PGURL="${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable}"
BROKER="tcp://127.0.0.1:1883"

cleanup() {
  docker rm -f ct-e2e-mosq >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "== 1. broker =="
if ! ss -ltn | grep -q ':1883 '; then
  docker run -d --name ct-e2e-mosq -p 1883:1883 eclipse-mosquitto:2 >/dev/null
  sleep 2
fi

echo "== 2. database =="
if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
  docker run -d --name ct-e2e-pg -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=transformers -p 5432:5432 postgres:16-alpine >/dev/null
  for _ in $(seq 1 30); do pg_isready -h localhost -p 5432 >/dev/null 2>&1 && break; sleep 1; done
fi
DATABASE_URL="$PGURL" go run ./cmd/dbmigrate up >/dev/null

echo "== 3. ingestion (postgres store) =="
go build -o /tmp/ct-e2e-ing ./cmd/ingestion
/tmp/ct-e2e-ing -broker "$BROKER" -store postgres >/tmp/ct-e2e-ing.log 2>&1 &
ING_PID=$!
sleep 2

echo "== 4. publish telemetry (simulator) =="
go run ./cmd/simulator -broker "$BROKER" -n 3 -ticks 5 -interval 1 -intensity 1.2 >/dev/null
sleep 1
kill -INT $ING_PID 2>/dev/null || true
wait $ING_PID 2>/dev/null || true

echo "== 5. assertions =="
RAW=$(psql "$PGURL" -tAc 'SELECT count(*) FROM raw_telemetry' | tr -d ' ')
MEAS=$(psql "$PGURL" -tAc 'SELECT count(*) FROM measurements' | tr -d ' ')
if [ "${RAW:-0}" -lt 15 ] || [ "${MEAS:-0}" -lt 15 ]; then
  echo "FAIL: expected >=15 rows, raw=$RAW meas=$MEAS"
  tail -5 /tmp/ct-e2e-ing.log
  exit 1
fi
echo "bronze OK: raw=$RAW measurements=$MEAS"

echo "== 6. dbt silver =="
(cd dbt && ../.venv/bin/dbt run --profiles-dir . --select stg_telemetry int_telemetry >/tmp/ct-e2e-dbt.log 2>&1)
STG=$(psql "$PGURL" -tAc 'SELECT count(*) FROM stg_telemetry' | tr -d ' ')
if [ "${STG:-0}" -lt 15 ]; then
  echo "FAIL: stg_telemetry=$STG (expected >=15)"
  tail -10 /tmp/ct-e2e-dbt.log
  exit 1
fi
echo "silver OK: stg_telemetry=$STG"
echo "E2E PASS"
