-- Silver data test: derived thermal stress index must live in [0,1].
select id, transformer_id, thermal_stress_index
from {{ ref('int_telemetry') }}
where thermal_stress_index < 0 or thermal_stress_index > 1
