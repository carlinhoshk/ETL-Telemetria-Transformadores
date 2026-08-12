-- +goose Up
-- Normalized telemetry measurements (operational hot path). The ingestion
-- service writes here; dedup/idempotency is enforced with the natural key.
CREATE TABLE measurements (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transformer_id       text             NOT NULL REFERENCES transformers (transformer_id),
    ts                   timestamptz      NOT NULL,
    load_percent         numeric(6,2)     NOT NULL CHECK (load_percent BETWEEN 0 AND 200),
    ambient_temperature_c numeric(5,2)    NOT NULL CHECK (ambient_temperature_c BETWEEN -20 AND 55),
    oil_temperature_c    numeric(5,2)     NOT NULL CHECK (oil_temperature_c BETWEEN -20 AND 150),
    winding_temperature_c numeric(5,2)    NOT NULL CHECK (winding_temperature_c BETWEEN -20 AND 200),
    oil_level_percent    numeric(6,2)     NOT NULL CHECK (oil_level_percent BETWEEN 0 AND 100),
    current_a            numeric(10,2)    NOT NULL CHECK (current_a >= 0),
    voltage_kv           numeric(8,2)     NOT NULL CHECK (voltage_kv > 0),
    state                text             NOT NULL CHECK (state IN ('NORMAL','WARNING','CRITICAL')),
    created_at           timestamptz      NOT NULL DEFAULT now(),
    -- Natural key for idempotent ingestion (QoS 1 redelivery).
    UNIQUE (transformer_id, ts)
);

CREATE INDEX idx_measurements_ts ON measurements (ts);
CREATE INDEX idx_measurements_transformer_ts ON measurements (transformer_id, ts DESC);

-- +goose Down
DROP TABLE IF EXISTS measurements;
