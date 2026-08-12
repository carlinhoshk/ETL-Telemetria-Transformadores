-- Silver data test: the payload state and the independent recomputation from
-- physical thresholds must agree (the Go ingestion normalizes state, so any
-- drift here points to a bug on either side).
select id, transformer_id, ts, state, state_recomputed
from {{ ref('int_telemetry') }}
where state is distinct from state_recomputed
