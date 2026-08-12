-- +goose Up
-- State/domain events: WARNING/CRITICAL transitions, maintenance-adjacent
-- notices, etc. Payload is kept as JSONB for flexible event shapes.
CREATE TABLE events (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transformer_id text        NOT NULL REFERENCES transformers (transformer_id),
    event_type     text        NOT NULL,
    severity       text        NOT NULL CHECK (severity IN ('INFO','WARNING','CRITICAL')),
    ts             timestamptz NOT NULL,
    payload        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_transformer_ts ON events (transformer_id, ts DESC);
CREATE INDEX idx_events_type ON events (event_type);

-- +goose Down
DROP TABLE IF EXISTS events;
