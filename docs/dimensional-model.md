# Dimensional Model (Gold, Phase 8)

The gold layer is a star schema for analytical queries — historical trends,
dashboards and the ML feature layer. Built by dbt from silver.

## Star schema

```text
          dim_time  ──────────┐
          dim_transformer ────┤
          dim_location  ──────┤  fact_transformer_measurement
          dim_sensor  (catalog)│
                              └── fact_transformer_event
```

### Dimensions

| Model | Grain | Notes |
|-------|-------|-------|
| `dim_time` | 1 row per distinct measurement instant | second/minute/hour/day/month/year, weekday, `is_weekend` |
| `dim_transformer` | 1 row per transformer | full design/project base — the similarity input features |
| `dim_location` | 1 row per transformer (1:1) | **synthetic** site attributes (deterministic hash of the id; substation/city/region/country) |
| `dim_sensor` | 1 row per sensor | catalog: type, unit, min/max limits (static) |

### Facts

| Fact | Grain | Measures |
|------|-------|----------|
| `fact_transformer_measurement` | (transformer, instant) | load, ambient/oil/winding temps, oil level, current, voltage, `thermal_stress_index`, state, temperature margins |
| `fact_transformer_event` | event | event_type, severity, count (currently empty — no event producers yet) |

## Why operational ≠ dimensional

- The **operational model** is normalized to keep the ingestion hot path
  cheap, the data integrity strong (FK + CHECK at write time) and the writes
  transactional (asset management, backfill).
- The **gold/dimensional** model is denormalized and **conformed** (shared
  dimension keys) so analytical SQL — drill-downs across time, transformers,
  locations and sensors — is simple and fast.
- **Silver** is the buffer: validation, cleansing, dedup and normalization
  happen there so the warehouse never sees dirty records. The star schema only
  references the curated layer.
- Full rationale: `docs/data-model.md`.

## Example analytical queries

Average thermal stress by region and application:

```sql
select dl.region, dt.application,
       avg(fm.thermal_stress_index) as avg_tsi,
       max(fm.winding_temperature_c) as max_winding_c
from fact_transformer_measurement fm
join dim_location dl      on dl.location_key      = fm.location_key
join dim_transformer dt   on dt.transformer_key   = fm.transformer_key
group by 1, 2
order by 3 desc;
```

Peak loads per weekday (dim_time drill-down):

```sql
select dt.weekday, max(fm.load_percent) as peak_load
from fact_transformer_measurement fm
join dim_time dt on dt.time_key = fm.time_key
group by 1 order by 2 desc;
```

## Run

```sh
make dbt        # full pipeline (seed + silver + gold + tests)
make dbt-gold   # gold only
```
