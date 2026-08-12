"""Shared fixtures for the ML service tests."""
from __future__ import annotations

import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def design_base() -> list[dict]:
    """A small deterministic design base (40-like rows, 10 units)."""
    rng = np.random.default_rng(7)
    base = []
    for i in range(10):
        mva = float(20 + rng.uniform(0, 120))
        hv = float(69 + rng.uniform(0, 161))
        base.append(
            {
                "transformer_id": f"TR-{i + 1:03d}",
                "rated_power_mva": mva,
                "hv_voltage_kv": hv,
                "lv_voltage_kv": 11.0 + rng.uniform(0, 58),
                "frequency_hz": 60.0 if i % 2 == 0 else 50.0,
                "phase_count": 3.0 if i % 4 != 3 else 1.0,
                "impedance_percent": 6.0 + rng.uniform(0, 20),
                "commissioning_year": float(1990 + rng.integers(0, 35)),
                "no_load_loss_kw": float(rng.uniform(10, 90)),
                "load_loss_kw": float(rng.uniform(100, 500)),
                "total_mass_t": float(rng.uniform(40, 200)),
            }
        )
    return base


@pytest.fixture
def telemetry_series() -> list[dict]:
    """~200 normal telemetry points plus one injected extreme reading."""
    rng = np.random.default_rng(11)
    rows = []
    for i in range(200):
        rows.append(
            {
                "transformer_id": "TR-001",
                "timestamp": f"2026-08-12T05:{i // 60:02d}:{i % 60:02d}Z",
                "load_percent": float(np.clip(rng.normal(65, 8), 40, 90)),
                "ambient_temperature_c": float(rng.normal(25, 3)),
                "oil_temperature_c": float(rng.normal(58, 4)),
                "winding_temperature_c": float(rng.normal(72, 5)),
                "oil_level_percent": float(rng.normal(95, 2)),
            }
        )
    # Injected anomaly: extreme winding temperature at fixed load.
    rows[150]["winding_temperature_c"] = 150.0
    rows[150]["load_percent"] = 98.0
    return rows
