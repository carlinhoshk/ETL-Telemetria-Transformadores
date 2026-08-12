-- Telemetry normalization and derived-field macros shared by silver/gold.

-- Extract a payload field as text (NULL when missing). Parenthesized so a
-- caller can safely cast: (payload ->> 'key')::numeric.
{% macro payload_field(payload, key) -%}
  ({{ payload }} ->> '{{ key }}')
{%- endmacro %}

-- Normalize an ISO-8601 (RFC3339) timestamp string to timestamptz, second
-- precision, preserving the payload instant.
{% macro normalize_ts(ts_expr) -%}
  cast(date_trunc('second', ({{ ts_expr }})::timestamptz) as timestamptz)
{%- endmacro %}

-- Thermal stress index in [0,1]: weighted blend of winding (vs critical 105°C),
-- oil (vs critical 100°C) and load (vs critical 140%). Higher = more stressed.
{% macro thermal_stress_index(winding_c, oil_c, load_pct) -%}
  round(least(1.0,
        0.5 * greatest({{ winding_c }}, 0) / {{ var('critical_winding_c') }}
      + 0.3 * greatest({{ oil_c }}, 0) / {{ var('critical_oil_c') }}
      + 0.2 * greatest({{ load_pct }}, 0) / {{ var('critical_load_percent') }}
    )::numeric, 3)
{%- endmacro %}

-- Recompute the operating state from physical thresholds (same rules as the
-- Go simulator/ingestion, exported from internal/telemetry/physics.go).
{% macro classify_state(winding_c, oil_c, load_pct) -%}
  case
    when {{ winding_c }} >= {{ var('critical_winding_c') }}
      or {{ oil_c }} >= {{ var('critical_oil_c') }}
      or {{ load_pct }} >= {{ var('critical_load_percent') }} then 'CRITICAL'
    when {{ winding_c }} >= 95.0
      or {{ oil_c }} >= 90.0
      or {{ load_pct }} > 100.0 then 'WARNING'
    else 'NORMAL'
  end
{%- endmacro %}
