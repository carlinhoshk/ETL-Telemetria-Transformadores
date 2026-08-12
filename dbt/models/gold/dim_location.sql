-- Gold: dim_location — synthetic site attributes. The synthetic fleet has no
-- real geography, so locations are derived deterministically from the
-- transformer id (hash-driven substation/city/region/country). Clearly fake
-- data, stable across runs.
{{ config(materialized='table') }}

with base as (
    select
        transformer_id,
        application,
        -- Deterministic index in [0,9] from the transformer id.
        abs(hashtext(transformer_id)) % 10 as loc_idx
    from {{ source('operational', 'transformers') }}
),

final as (
    select
        transformer_id as location_key,
        'SUB-' || upper(substr(transformer_id, 4, 3)) || '-' || loc_idx as substation,
        (ARRAY['São Paulo','Campinas','Jundiaí','Salto','Itu','Sorocaba','Piracicaba','Americana','Valinhos','Indaiatuba'])[loc_idx + 1] as city,
        case
            when application = 'generation' then 'Sul'
            when application = 'renewable' then 'Nordeste'
            when application = 'industrial' then 'Sudeste'
            else 'Sudeste'
        end as region,
        'Brazil' as country
    from base
)

select * from final
