-- Silver data test: the transformer_id declared inside the payload must match
-- the topic-level id that the ingestion service recorded (integrity check).
select id, transformer_id, payload_transformer_id
from {{ ref('stg_telemetry') }}
where transformer_id is distinct from payload_transformer_id
