-- +goose Up
-- Maintenance records for the transformer asset lifecycle.
CREATE TABLE maintenance (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transformer_id text        NOT NULL REFERENCES transformers (transformer_id),
    maintained_at  timestamptz NOT NULL,
    kind           text        NOT NULL,
    description    text        NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_maintenance_transformer ON maintenance (transformer_id, maintained_at DESC);

-- +goose Down
DROP TABLE IF EXISTS maintenance;
