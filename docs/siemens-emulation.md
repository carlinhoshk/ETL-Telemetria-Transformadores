# Siemens Energy Emulation Mapping

> Educational mapping between publicly documented Siemens Energy monitoring
> architectures and the components of this project. For reference only —
> **nothing proprietary is reproduced**. All data here is synthetic.

## Real systems surveyed (public sources)

### 1. Noedra — "Mind of the Grid" framework

Noedra is Siemens Energy's digital framework connecting Grid Technologies'
intelligent solutions (sensing, software, advisory) into one ecosystem. Each
suite addresses part of the grid:

- **Node** — substation digitalization: monitoring, diagnostics, inspection
  and analytics across transformers and switchgear (continuous awareness,
  predictive intelligence, automated control).
- **Flow** — overhead line inspections / dynamic ratings.
- **Shield** — OT/ICS cybersecurity.
- **Atlas** — additional grid intelligence suites.

Transformer digitalization under Node offers: operational and condition
monitoring, thermal/insulation/bushing/OLTC diagnostics, inspection
evidence management, and analytics (health, risk, aging, predictive
maintenance).

### 2. Sensformer / Sensformer Advanced

An IoT-based digital twin for transformers (launched 2018):

- An IoT gateway installed in the transformer's control cabinet measures
  physical signals (load, top-oil temperature, winding temperature, ambient
  temperature, oil level, DGA gases) and sends them to the cloud over
  GSM/LAN.
- The digital twin is built from the transformer's **design data** and uses a
  thermo-hydraulic model to provide virtual sensors, loss-of-life
  calculation, overload prediction, load-cycle simulation and fleet
  management.

### 3. SITRAM family (TDCM, Multisense, CAM, H2Guard)

- **SITRAM TDCM** — Transformer Diagnostic & Condition Management: online
  transformer data, early-fault diagnostics, prognoses and recommendations;
  works on transformers from any manufacturer and any age.
- **SITRAM Multisense** — online monitoring hardware/software (5, 9).
- **SITRAM CAM** — collection, visualization and analysis of transmission
  asset data for condition assessments.
- **SITRAM H2Guard** — hydrogen-in-oil gas monitoring.
- In 2026 Siemens Energy announced the acquisition of **Camlin Group**, a
  maker of sensor-based transformer monitoring software.

### 4. EnergyAI

`energyai.siemens-energy.cloud` — an internal platform for data analysis and
visualization.

### 5. Connected Factory (manufacturing IoT)

Siemens Energy standardized manufacturing asset data across factories using
AWS IoT SiteWise + S3 + Amazon Managed Grafana (their Jundiaí complex
manufactures transformers, turbines, capacitors and MV/HV equipment).

### 6. Design reuse & similarity

Siemens publicly documents the value of reusing engineering designs and
applies knowledge-graph technology (e.g., metaphactory on Amazon Neptune) to
manage a fleet and surface similar configurations/parts. This is the
conceptual basis of the job's **"mechanism of similarity between projects"**:
find past transformer design projects similar to a new one to accelerate
proposals/engineering.

## Mapping to this project

| Siemens Energy (real) | Role | This project |
|---|---|---|
| Sensformer IoT gateway / SITRAM Multisense (sensors: load, top-oil, winding, ambient) | Physical signal acquisition | Go **simulator** (synthetic, physics-plausible telemetry) |
| Sensformer Advanced digital twin (thermo-hydraulic model) | Digital twin: loss-of-life, overload, virtual sensors | Simulator physics (load→temperatures) + derived `thermal_stress_index` in dbt |
| MQTT / GSM-LAN gateway → cloud | Telemetry transport | MQTT (Eclipse Mosquitto) on `transformers/{id}/telemetry` |
| SITRAM CAM / EnergyAI / Noedra analytics | Collection, analytics, condition assessment | PostgreSQL + dbt silver/gold + Python ML + Go API |
| Connected Factory (AWS IoT → S3 → Grafana) | Industrial IoT platform | Dockerized local pipeline; documented cloud mapping (Phase 16) |
| Knowledge graph / design reuse | Similarity between projects/assets | **Similarity engine** (scikit-learn baseline, Phase 10) |
| Snowflake/Databricks (desirable in job profile) | Cloud warehouse / ELT | dbt + PostgreSQL medallion model (local emulation) |
| Noedra ecosystem | Product platform concept | Multi-component architecture with clear contracts |

## What we intentionally do NOT copy

- Internal schemas, proprietary algorithms or confidential design values.
- Branded product features beyond conceptual inspiration.
- Real manufacturer/utility data — everything is synthetic.

## Sources (public)

- Siemens Energy — Substation digitalization / Noedra Node suite
  (siemens-energy.com)
- "Sensformer: Powering the future with digitalized transformers",
  Transformers Magazine (2023) — public paper
- Siemens Energy — SITRAM TDCM / Multisense manuals & flyers (public PDFs)
- Siemens Energy press & case studies (Connected Factory / AWS; turbine
  knowledge graph with metaphactory)