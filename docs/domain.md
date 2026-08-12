# Transformer Domain

## Purpose

The domain package owns the core transformer concepts. The central record is
`domain.Transformer`: the design/project data of a power transformer. This
"historical project base" is what the job profile calls *bases históricas de
projetos de transformadores* and is the input for the project-similarity
mechanism (Phase 10).

## Record fields

Core design data:

- `transformer_id` (TR-XXX), `rated_power_mva`, `hv_voltage_kv`,
  `lv_voltage_kv`, `frequency_hz` (50/60), `phase_count` (1/3),
  `vector_group`, `impedance_percent`, `cooling_type`,
  `commissioning_year`, `application`.

Design extras (used for a richer similarity and engineering context):

- `no_load_loss_kw`, `load_loss_kw`, `total_mass_t`, `length_m`,
  `width_m`, `height_m`.

Categorical enums are explicit types: `Application`, `CoolingType`,
`VectorGroup`, each with a validated list.

## Validation

`Transformer.Validate()` enforces:

- ID format `TR-\d{3}`.
- Positive power; `hv_voltage_kv > lv_voltage_kv > 0`.
- Frequency 50 or 60 Hz; phase count 1 or 3.
- Known vector group and cooling type.
- Impedance in `(0, 40]` percent.
- Commissioning year from 1960 to today.
- Non-negative losses; positive mass and dimensions.

## Synthetic generator

`Generator` produces a fleet deterministically from a seed
(`NewGenerator(seed)`). Values are not purely random; they follow
engineering plausibility rules:

- Power is log-uniform within a per-application envelope
  (generation/transmission up to ~400 MVA, down to 5 MVA distribution).
- Voltage pairs come from real class tables (`69/115/138/230/345/500 kV`);
  very high-voltage classes are only allowed for large units
  (≥ 345 kV requires ≥ 80 MVA; ≥ 230 kV requires ≥ 40 MVA), and `lv` is
  always below `hv`.
- Impedance grows with power (`5 + 5·log10(MVA)`) with jitter.
- Cooling follows power: ONAN → ONAF → OFAF/ODAF/OFWF.
- Vector group matches the application (GSU/transmission tend to YNd;
  distribution/industrial tend to Dyn/Dd/Yy).
- Losses scale with power (no-load ≈ 0.08–0.16 %, load ≈ 0.6–1.2 %),
  mass and dimensions grow monotonically with power.
- Commissioning year is recent-weighted (1990–2025).

## Fleet artifacts

- `cmd/etl generate` writes the fleet as JSON or CSV
  (e.g. `go run ./cmd/etl generate -n 40 -seed 42 -out dbt/seeds/transformers.csv`).
- `dbt/seeds/transformers.csv` is the committed synthetic historical project
  base (seed 42), which dbt seeds from Phase 7 onward; the transformer design
  record is the source of `dim_transformer`.
- Later phases load this base into PostgreSQL (Phase 5) and read it from
  there; the ML service consumes it for project similarity (Phase 10).