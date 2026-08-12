"""Project similarity baseline (Phase 10).

Baseline, no LLM: normalized Euclidean distance over the scaled design
feature vectors. A candidate closer to the target in the standardized
feature space gets a higher similarity score in [0,1].
"""
from __future__ import annotations

from typing import Mapping, Sequence

import numpy as np

from .features import FeatureModel


def similarity_scores(
    target: Mapping[str, float],
    candidates: Sequence[Mapping[str, float]],
    model: FeatureModel,
) -> list[tuple[str, float]]:
    """Return (transformer_id, score) pairs, best first.

    The target must be excluded from candidates by the caller (the API
    filters it out). Scores are in [0,1]; 1.0 = identical feature vector.
    """
    tv = model.transform(target)
    scores: list[tuple[str, float]] = []
    for cand in candidates:
        cv = model.transform(cand)
        dist = float(np.linalg.norm(tv - cv))
        score = 1.0 / (1.0 + dist)  # monotonic: distance -> 0 gives 1.0
        scores.append((str(cand.get("transformer_id", "")), round(score, 4)))
    scores.sort(key=lambda item: item[1], reverse=True)
    return scores
