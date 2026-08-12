"""Anomaly detection over telemetry (Phase 9).

Baseline: IsolationForest on the scaled telemetry features
(load, ambient/oil/winding temperatures, oil level). Returns per-measurement
anomaly labels and scores (negative = more anomalous).
"""
from __future__ import annotations

from typing import Mapping, Sequence

import numpy as np
from sklearn.ensemble import IsolationForest

from .features import TELEMETRY_FEATURES, telemetry_matrix


def detect_anomalies(
    measurements: Sequence[Mapping[str, object]],
    contamination: float = 0.1,
    seed: int = 42,
) -> list[dict]:
    """Flag anomalous measurements.

    Returns one dict per input row with transformer_id, timestamp,
    anomaly (bool) and score (float).
    """
    if len(measurements) == 0:
        return []

    x = telemetry_matrix(measurements)  # type: ignore[arg-type]
    model = IsolationForest(contamination=contamination, random_state=seed)
    labels = model.fit_predict(x)  # -1 = anomaly, 1 = normal
    scores = model.score_samples(x)  # higher = more normal

    out: list[dict] = []
    for i, row in enumerate(measurements):
        out.append(
            {
                "transformer_id": row.get("transformer_id"),
                "timestamp": row.get("timestamp"),
                "anomaly": bool(labels[i] == -1),
                "score": round(float(scores[i]), 4),
            }
        )
    return out
