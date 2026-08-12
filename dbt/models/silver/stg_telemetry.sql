-- Silver: staged telemetry extracted from the raw bronze payload.
-- Validation, deduplication and timestamp normalization happen here.
-- The raw record carries the original message as JSONB; we extract typed
-- columns, keep the payload-declared transformer id for an integrity test,
-- and deduplicate on the natural key (transformer_id, payload timestamp).
{{ config(materialized='table') }}

with source as (
    select
        id,
        transformer_id,
        schema_version,
        topic,
        source,
        ingested_at,
        payload
    from {{ source('operational', 'raw_telemetry') }}
    where schema_version = 1
),

extracted as (
    select
        id,
        transformer_id,
        {{ payload_field('payload', 'transformer_id') }} as payload_transformer_id,
        {{ payload_field('payload', 'timestamp') }} as ts_raw,
        source,
        ingested_at,
        {{ payload_field('payload', 'load_percent') }}::numeric as load_percent,
        {{ payload_field('payload', 'ambient_temperature_c') }}::numeric as ambient_temperature_c,
        {{ payload_field('payload', 'oil_temperature_c') }}::numeric as oil_temperature_c,
        {{ payload_field('payload', 'winding_temperature_c') }}::numeric as winding_temperature_c,
        {{ payload_field('payload', 'oil_level_percent') }}::numeric as oil_level_percent,
        {{ payload_field('payload', 'current_a') }}::numeric as current_a,
        {{ payload_field('payload', 'voltage_kv') }}::numeric as voltage_kv,
        {{ payload_field('payload', 'state') }} as state_payload,
        -- Missing value handling: keep NULLs, flag the row for downstream QA.
        case
            when {{ payload_field('payload', 'load_percent') }} is null
              or {{ payload_field('payload', 'winding_temperature_c') }} is null
              or {{ payload_field('payload', 'oil_temperature_c') }} is null
            then 'incomplete'
            else 'complete'
        end as quality_flag
    from source
),

normalized as (
    select
        *,
        -- Dedup: same transformer + payload timestamp, keep the most recent
        -- ingestion (MQTT QoS 1 redelivery may produce identical keys).
        row_number() over (
            partition by transformer_id, ts_raw
            order by ingested_at desc, id desc
        ) as rn
    from extracted
),

final as (
    select
        id,
        transformer_id,
        payload_transformer_id,
        source,
        ingested_at,
        {{ normalize_ts('ts_raw') }} as ts,
        load_percent,
        ambient_temperature_c,
        oil_temperature_c,
        winding_temperature_c,
        oil_level_percent,
        current_a,
        voltage_kv,
        upper(trim(state_payload)) as state,
        quality_flag
    from normalized
    where rn = 1
)

select * from final
