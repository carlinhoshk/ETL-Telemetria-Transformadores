# Python ML Service (Phase 9)

A separate, stateless Python service for ML compute: feature preparation,
scaling, **similarity** and **anomaly** detection. It has no database
access — the Go API gathers the data and calls it over HTTP, keeping the
ML service a pure compute component (contracts well defined).

## Stack

- Python 3.11, NumPy, Pandas, scikit-learn.
- HTTP: standard library `http.server` (no framework dependency).
- Tests: pytest.

## Modules (`python/ml_service/`)

| Module       | Responsibility                                            |
|--------------|-----------------------------------------------------------|
| `features.py`| `fit_design_features` (StandardScaler over design features), `telemetry_matrix` |
| `similarity.py` | normalized-Euclidean similarity baseline (Phase 10)    |
| `anomaly.py` | IsolationForest over telemetry features                   |
| `service.py` | JSON HTTP server (`GET /health`, `POST /similar`, `POST /anomaly`) |

Design features (the similarity input): `rated_power_mva`, `hv_voltage_kv`,
`lv_voltage_kv`, `frequency_hz`, `phase_count`, `impedance_percent`,
`commissioning_year`, `no_load_loss_kw`, `load_loss_kw`, `total_mass_t`.

Anomaly features: `load_percent`, `ambient_temperature_c`,
`oil_temperature_c`, `winding_temperature_c`, `oil_level_percent`.

## API

```sh
python -m ml_service --host 127.0.0.1 --port 8081
```

- `GET /health` → `{"status":"ok"}`
- `POST /similar` — body `{target, candidates, top_k?}` →
  `{results:[{transformer_id, score}], features:[...]}`
- `POST /anomaly` — body `{measurements, contamination?}` →
  `{results:[{transformer_id, timestamp, anomaly, score}]}`

## Tests

```sh
make ml-test
# PYTHONPATH=python .venv/bin/pytest python/ml_service/tests
```

Covers feature scaling invariants (zero mean, unit variance on varied
features), similarity ordering/bounds/self-match, injected-outlier
detection, and the HTTP routes end to end.
