-- Gold: dim_transformer — conformed design dimension. The design/project
-- attributes are the input features for the similarity mechanism (Phase 10),
-- so this dimension is deliberately stable and complete.
{{ config(materialized='table') }}

select
    transformer_id as transformer_key,
    rated_power_mva,
    hv_voltage_kv,
    lv_voltage_kv,
    frequency_hz,
    phase_count,
    vector_group,
    impedance_percent,
    cooling_type,
    commissioning_year,
    application,
    no_load_loss_kw,
    load_loss_kw,
    total_mass_t,
    length_m,
    width_m,
    height_m
from {{ source('operational', 'transformers') }}
