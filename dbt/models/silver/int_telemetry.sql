-- Silver: enriched measurements with derived fields. This is the curated
-- layer trusted by analytics and the ML feature layer: one row per unique
-- (transformer, instant) with quality flags and derived thermal metrics.
{{ config(materialized='table') }}

with stg as (
    select * from {{ ref('stg_telemetry') }}
),

enriched as (
    select
        stg.id,
        stg.transformer_id,
        stg.ts,
        stg.source,
        stg.ingested_at,
        stg.load_percent,
        stg.ambient_temperature_c,
        stg.oil_temperature_c,
        stg.winding_temperature_c,
        stg.oil_level_percent,
        stg.current_a,
        stg.voltage_kv,
        stg.state,
        -- Derived field: thermal stress index in [0,1].
        {{ thermal_stress_index(
            'stg.winding_temperature_c',
            'stg.oil_temperature_c',
            'stg.load_percent') }} as thermal_stress_index,
        -- Independent recomputation of the state (cross-check vs payload).
        {{ classify_state(
            'stg.winding_temperature_c',
            'stg.oil_temperature_c',
            'stg.load_percent') }} as state_recomputed,
        -- Margins to alarm thresholds (engineering insight).
        round(({{ var('critical_winding_c') }} - stg.winding_temperature_c)::numeric, 2) as winding_margin_c,
        round(({{ var('critical_oil_c') }} - stg.oil_temperature_c)::numeric, 2) as oil_margin_c,
        stg.quality_flag
    from stg
)

select * from enriched
