-- +goose Up
-- Operational (OLTP) model, Phase 5. Design/project base of historical
-- transformers — the input for the similarity mechanism (Phase 10).
CREATE TABLE transformers (
    transformer_id      text PRIMARY KEY,
    rated_power_mva     numeric(8,2)     NOT NULL CHECK (rated_power_mva > 0),
    hv_voltage_kv       numeric(8,2)     NOT NULL CHECK (hv_voltage_kv > 0),
    lv_voltage_kv       numeric(8,2)     NOT NULL CHECK (lv_voltage_kv > 0),
    frequency_hz        integer          NOT NULL CHECK (frequency_hz IN (50, 60)),
    phase_count         integer          NOT NULL CHECK (phase_count IN (1, 3)),
    vector_group        text             NOT NULL,
    impedance_percent   numeric(5,2)     NOT NULL CHECK (impedance_percent > 0 AND impedance_percent <= 40),
    cooling_type        text             NOT NULL,
    commissioning_year  integer          NOT NULL CHECK (commissioning_year BETWEEN 1960 AND 2100),
    application         text             NOT NULL,
    no_load_loss_kw     numeric(9,2)     NOT NULL CHECK (no_load_loss_kw >= 0),
    load_loss_kw        numeric(9,2)     NOT NULL CHECK (load_loss_kw >= 0),
    total_mass_t        numeric(9,2)     NOT NULL CHECK (total_mass_t > 0),
    length_m            numeric(6,2)     NOT NULL CHECK (length_m > 0),
    width_m             numeric(6,2)     NOT NULL CHECK (width_m > 0),
    height_m            numeric(6,2)     NOT NULL CHECK (height_m > 0),
    created_at          timestamptz      NOT NULL DEFAULT now(),
    updated_at          timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX idx_transformers_application ON transformers (application);
CREATE INDEX idx_transformers_cooling ON transformers (cooling_type);

-- +goose Down
DROP TABLE IF EXISTS transformers;
