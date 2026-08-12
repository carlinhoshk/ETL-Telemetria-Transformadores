from ml_service.anomaly import detect_anomalies


def test_no_anomalies_on_empty(telemetry_series):
    assert detect_anomalies([]) == []


def test_injected_outlier_is_flagged(telemetry_series):
    results = detect_anomalies(telemetry_series, contamination=0.05, seed=42)
    flagged = {r["timestamp"] for r in results if r["anomaly"]}
    assert telemetry_series[150]["timestamp"] in flagged, "injected outlier not caught"


def test_scores_and_shape(telemetry_series):
    results = detect_anomalies(telemetry_series, contamination=0.1, seed=42)
    assert len(results) == len(telemetry_series)
    for r in results:
        assert r["transformer_id"] == "TR-001"
        assert isinstance(r["anomaly"], bool)
        assert isinstance(r["score"], float)
