# Transformer Project Similarity (Phase 10)

The similarity mechanism finds transformers whose **design characteristics**
resemble a target unit. It is a baseline approach — deliberately no LLM —
built on the historical project base (Phase 1) and exposed through the API
(`GET /transformers/{id}/similar`, Phase 11).

## Method

1. **Features** — numeric design columns: rated power, HV/LV voltages,
   frequency, phase count, impedance, commissioning year, losses and mass.
2. **Scaling** — a `StandardScaler` is fit over the candidate base so
   different units (MW vs % vs years) become comparable.
3. **Distance** — Euclidean distance in the standardized feature space.
4. **Score** — `similarity = 1 / (1 + distance)`, in `(0, 1]`; higher is
   closer. Identical feature vectors score 1.0.

The target is excluded from its own candidate list by the API.

## Why this baseline

- Explainable, deterministic and cheap — good for a portfolio and for
  downstream systems that need stable scores.
- The feature set is the project design base the business already curates.
- The pipeline (fit → transform → distance → score) is the same shape a more
  sophisticated model (e.g. learned embeddings) would plug into.

## Where it lives

- Algorithm: `python/ml_service/similarity.py`.
- HTTP route: `POST /similar` on the ML service (Phase 9).
- API exposure: `GET /transformers/{id}/similar` on the Go API (Phase 11).

## Example

For `TR-001` (a 35.6 MVA / 115 kV generation unit) the top-3 similar units
found in the synthetic base were `TR-018`, `TR-039`, `TR-033` with scores
≈ 0.60 / 0.58 / 0.55.

## Known limitations

- Similarity is univariate-distance based; categorical design attributes
  (vector group, cooling type) are not encoded yet.
- Scores are relative to the current candidate base; adding units changes
  the scaling and thus the absolute scores.
