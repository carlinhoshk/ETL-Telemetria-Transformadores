"""Shared bootstrap helpers for the portfolio notebooks.

Thin notebook policy (AGENTS.md principle 4): the notebooks reuse the
actual project modules (`python/ml_service`) and services (Go API, ML
service) instead of reimplementing the pipeline.
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import pandas as pd

# Make the ML service importable as `ml_service` from the notebooks.
_HERE = Path(__file__).resolve().parent
_PROJECT = _HERE.parent
for _p in (_PROJECT / "python", str(_PROJECT)):
    if _p not in sys.path:
        sys.path.insert(0, str(_p))

CSV_PATH = _PROJECT / "dbt" / "seeds" / "transformers.csv"

# Base URLs of the locally running services (make demo / make ml-run / make api).
ML_URL = os.getenv("ML_URL", "http://localhost:8081")
API_URL = os.getenv("API_URL", "http://localhost:8080")

# Default local Postgres connection (override with TEST_DATABASE_URL).
DSN = os.getenv(
    "TEST_DATABASE_URL",
    "postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable",
)


def load_design_base() -> pd.DataFrame:
    """Load the historical design base (40 synthetic transformer projects)."""
    return pd.read_csv(CSV_PATH)


def pg_connect():
    """Return a psycopg2 connection to the local PostgreSQL."""
    import psycopg2

    return psycopg2.connect(DSN)


def pg_df(query: str, params: tuple | None = None) -> pd.DataFrame:
    """Run a SQL query against PostgreSQL and return a DataFrame."""
    import psycopg2
    import warnings

    with psycopg2.connect(DSN) as conn:
        with warnings.catch_warnings():
            warnings.filterwarnings(
                "ignore",
                message="pandas only supports SQLAlchemy connectable.*",
            )
            return pd.read_sql_query(query, conn, params=params)


def to_plain_records(df: pd.DataFrame) -> list[dict]:
    """Convert a DataFrame to JSON-serializable records.

    Timestamps (e.g. created_at/updated_at from the operational model)
    are converted to ISO 8601 strings so the records can be sent to the
    ML service and Go API over HTTP.
    """
    out = []
    for row in df.to_dict("records"):
        out.append(
            {
                k: (v.isoformat() if isinstance(v, pd.Timestamp) else v)
                for k, v in row.items()
            }
        )
    return out


def design_features() -> list[str]:
    """Design features used by the similarity mechanism (docs/similarity.md)."""
    from ml_service.features import DESIGN_FEATURES

    return list(DESIGN_FEATURES)


def similar_for(target_id: str, top_k: int = 5) -> pd.DataFrame:
    """Similarity results for a transformer ID from the ML service."""
    import requests

    base = load_design_base()
    target = base[base["transformer_id"] == target_id].iloc[0].to_dict()
    candidates = [
        row.to_dict()
        for _, row in base[base["transformer_id"] != target_id].iterrows()
    ]
    resp = requests.post(
        f"{ML_URL}/similar",
        json={"target": target, "candidates": candidates, "top_k": top_k},
        timeout=15,
    )
    resp.raise_for_status()
    return pd.DataFrame(resp.json()["results"])