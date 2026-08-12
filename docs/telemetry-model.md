# Telemetry Model (Simulator)

The Go simulator (`internal/telemetry`, `cmd/simulator`) produces one
`Measurement` per transformer per tick. Values follow physics-*plausible*
relationships — inspired by transformer thermal models (top-oil and
hot-spot) — and are deterministic for a fixed seed. Nothing is purely random.

## Emitted fields (per tick)

- `timestamp` (ISO-8601 UTC), `transformer_id`, `schema_version`
- `load_percent` — daily load shape, scaled by configurable intensity
- `ambient_temperature_c` — daily cycle with a sine around `14:00`
- `oil_temperature_c` — top-oil temperature (thermal lag model)
- `winding_temperature_c` — hot-spot / winding temperature (faster lag)
- `oil_level_percent` — expands slightly with temperature
- `current_a` — derived from apparent power: `I = S/(√3·V)` on the HV side
- `voltage_kv` — nominal HV with a small droop under load
- `state` — NORMAL / WARNING / CRITICAL

## Physics rules

**Load** (`targetLoad`): a composite daily profile (morning plateau + evening
peak) with a per-unit time offset, weather coupling and ±3 % noise, bounded
to 5–160 %. `load_intensity` scales it: 1.0 nominal, >1.0 pushes units into
overload.

**Top-oil temperature** (steady state, classic model):

```
oil_steady = ambient + max_oil_rise · ((1 + R·K²)/(1 + R))^0.8
```

where `K = load_percent/100` and `R = load_loss / no_load_loss` (taken from
each transformer's design record). `max_oil_rise` depends on cooling type
(48 °C ONAN … 60 °C ODAF) plus a per-unit jitter.

**Winding (hot-spot) temperature**:

```
winding_steady = oil + winding_grad · K^1.6
```

with `winding_grad` per cooling type (20 °C ONAN … 12 °C ODAF) plus jitter.

**Thermal inertia**: temperatures converge toward steady state through a
first-order lag — oil time constant ~90 min, winding ~7 min. Load steps never
jump temperatures instantly; the envelope reaches steady state slowly, which
is exactly why a fleet under sustained overload drifts into WARNING/CRITICAL
over hours.

**Relations that always hold:**
- load ↑ ⇒ winding ↑ ⇒ oil ↑
- ambient ↑ ⇒ oil ↑ (and steady-state rise is over ambient)
- `winding_temperature_c ≥ oil_temperature_c` in normal operation
- `oil_level` stays in 85–99.8 %; `voltage` droops slightly with load

## States

| State | Trigger |
|---|---|
| NORMAL | winding < 95 °C, oil < 90 °C, load ≤ 100 % |
| WARNING | winding ≥ 95 °C or oil ≥ 90 °C or load > 100 % |
| CRITICAL | winding ≥ 105 °C or oil ≥ 100 °C or load ≥ 140 % |

Thresholds live in `internal/telemetry/physics.go` (`WarningWindingC`, ...).

## Determinism

Everything stochastic comes from a seedable `math/rand` source: per-unit
profile offsets, small jitters and noise. Given the same seed, interval,
intensity and start time, the same fleet produces identical telemetry.

## CLI

```
go run ./cmd/simulator -n 4 -interval 5 -seed 42 -intensity 1.0 -ticks 3
```

Flags: `-csv` (fleet design base), `-n` (units), `-interval` (s), `-seed`,
`-intensity`, `-ticks`. Output is JSON Lines to stdout. Phase 3 streams the
same measurements over MQTT.