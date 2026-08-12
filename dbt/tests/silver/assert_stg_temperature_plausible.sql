-- Silver data test: temperatures must be physically plausible and the
-- winding must always be at least as hot as the oil.
select id, transformer_id, winding_temperature_c, oil_temperature_c
from {{ ref('stg_telemetry') }}
where winding_temperature_c < oil_temperature_c
   or winding_temperature_c > 200
   or oil_temperature_c > 150
   or ambient_temperature_c < -20
   or ambient_temperature_c > 55
