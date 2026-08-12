"""Feature preparation and scaling for the transformer design base.

The design features are the numeric columns of the historical project base
(Phase 1). They feed the similarity mechanism and the anomaly detector.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Mapping, Sequence

import numpy as np
import pandas as pd
from sklearn.preprocessing import StandardScaler

# Numeric design features used for similarity (see docs/similarity.md).
DESIGN_FEATURES: tuple[str, ...] = (
    "rated_power_mva",
    "hv_voltage_kv",
    "lv_voltage_kv",
    "frequency_hz",
    "phase_count",
    "impedance_percent",
    "commissioning_year",
    "no_load_loss_kw",
    "load_loss_kw",
    "total_mass_t",
)

# Telemetry features used by the anomaly detector.
TELEMETRY_FEATURES: tuple[str, ...] = (
    "load_percent",
    "ambient_temperature_c",
    "oil_temperature_c",
    "winding_temperature_c",
    "oil_level_percent",
)


@dataclass(frozen=True)
class FeatureModel:
    """A fitted scaler plus the ordered feature columns it expects."""

    scaler: StandardScaler
    columns: tuple[str, ...]

    def transform(self, row: Mapping[str, float]) -> np.ndarray:
        """Scale a single design row into a feature vector."""
        frame = pd.DataFrame([{c: row.get(c, 0.0) for c in self.columns}])
        return self.scaler.transform(frame)[0]


def fit_design_features(rows: Sequence[Mapping[str, float]]) -> FeatureModel:
    """Fit a StandardScaler over the design features of the candidate base."""
    frame = pd.DataFrame([{c: r.get(c, 0.0) for c in DESIGN_FEATURES} for r in rows])
    scaler = StandardScaler()
    scaler.fit(frame.values)
    return FeatureModel(scaler=scaler, columns=tuple(DESIGN_FEATURES))


def telemetry_matrix(measurements: Sequence[Mapping[str, float]]) -> np.ndarray:
    """Build the numeric telemetry matrix used by the anomaly detector."""
    frame = pd.DataFrame(
        [{c: float(m.get(c, 0.0)) for c in TELEMETRY_FEATURES} for m in measurements]
    )
    return frame.values
