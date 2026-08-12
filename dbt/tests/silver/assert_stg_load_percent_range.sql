-- Silver data test: load_percent must stay in the physical envelope [0,200].
select id, transformer_id, load_percent
from {{ ref('stg_telemetry') }}
where load_percent < 0 or load_percent > 200
