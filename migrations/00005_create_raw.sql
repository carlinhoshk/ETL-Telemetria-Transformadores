-- +goose Up
-- Bronze layer (Phase 6): append-only, faithful capture of raw events for
-- replay and audit. Original payload preserved verbatim; nothing mutated.
CREATE TABLE raw_telemetry (
    id              text PRIMARY KEY,          -- dedup key {transformer_id}@{timestamp}
    transformer_id  text        NOT NULL REFERENCES transformers (transformer_id),
    schema_version  integer     NOT NULL,
    topic           text        NOT NULL,
    source          text        NOT NULL,
    ingested_at     timestamptz NOT NULL DEFAULT now(),
    payload         jsonb       NOT NULL
);

CREATE INDEX idx_raw_telemetry_transformer ON raw_telemetry (transformer_id, ingested_at DESC);
CREATE INDEX idx_raw_telemetry_ingested ON raw_telemetry (ingested_at);

CREATE TABLE raw_events (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transformer_id  text        NOT NULL REFERENCES transformers (transformer_id),
    event_type      text        NOT NULL,
    severity        text        NOT NULL,
    topic           text        NOT NULL,
    source          text        NOT NULL,
    ingested_at     timestamptz NOT NULL DEFAULT now(),
    payload         jsonb       NOT NULL
);

CREATE INDEX idx_raw_events_transformer ON raw_events (transformer_id, ingested_at DESC);

-- +goose Down
DROP TABLE IF EXISTS raw_events;
DROP TABLE IF EXISTS raw_telemetry;
