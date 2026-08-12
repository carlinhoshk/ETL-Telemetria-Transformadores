-- Gold: fact_transformer_measurement — the analytical fact table. Wide format:
-- one row per unique (transformer, instant) with all sensor measures and the
-- derived thermal stress index. Joins the conformed dimensions.
{{ config(materialized='table') }}

with m as (
    select *
    from {{ ref('int_telemetry') }}
)

select
    m.ts as time_key,
    m.transformer_id as transformer_key,
    m.transformer_id as location_key,
    m.load_percent,
    m.ambient_temperature_c,
    m.oil_temperature_c,
    m.winding_temperature_c,
    m.oil_level_percent,
    m.current_a,
    m.voltage_kv,
    m.thermal_stress_index,
    m.state,
    m.winding_margin_c,
    m.oil_margin_c
from m
