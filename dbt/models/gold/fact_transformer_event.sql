-- Gold: fact_transformer_event — event facts (severity + state). The
-- operational events table is currently empty (the simulator emits
-- measurements, not events), so this fact stays empty until event producers
-- exist, but the star-schema shape is conformed from day one.
{{ config(materialized='table') }}

with e as (
    select
        transformer_id,
        ts,
        event_type,
        severity
    from {{ source('operational', 'events') }}
)

select
    e.ts as time_key,
    e.transformer_id as transformer_key,
    e.transformer_id as location_key,
    e.event_type,
    e.severity,
    count(*) over (partition by e.transformer_id, e.event_type, e.severity) as event_count
from e
