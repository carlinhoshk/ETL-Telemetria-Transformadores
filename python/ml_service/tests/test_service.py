"""End-to-end test of the HTTP service (similarity + anomaly routes)."""
from __future__ import annotations

import json
import threading
from http.client import HTTPConnection
from http.server import ThreadingHTTPServer

from ml_service.service import MLHandler


def _serve():
    server = ThreadingHTTPServer(("127.0.0.1", 0), MLHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    host, port = server.server_address
    return server, f"127.0.0.1:{port}"


def _post(addr: str, path: str, body: dict):
    conn = HTTPConnection(addr)
    conn.request(
        "POST",
        path,
        body=json.dumps(body),
        headers={"Content-Type": "application/json"},
    )
    resp = conn.getresponse()
    return resp.status, json.loads(resp.read())


def _get(addr: str, path: str):
    conn = HTTPConnection(addr)
    conn.request("GET", path)
    resp = conn.getresponse()
    return resp.status, json.loads(resp.read())


def test_health(design_base):
    server, addr = _serve()
    try:
        status, body = _get(addr, "/health")
        assert status == 200
        assert body["status"] == "ok"
    finally:
        server.shutdown()


def test_similar_route(design_base):
    server, addr = _serve()
    try:
        target = design_base[0]
        candidates = [r for r in design_base if r["transformer_id"] != target["transformer_id"]]
        status, body = _post(addr, "/similar", {"target": target, "candidates": candidates, "top_k": 3})
        assert status == 200
        assert len(body["results"]) == 3
        assert all({"transformer_id", "score"} <= set(r) for r in body["results"])
        scores = [r["score"] for r in body["results"]]
        assert scores == sorted(scores, reverse=True)
    finally:
        server.shutdown()


def test_anomaly_route(telemetry_series):
    server, addr = _serve()
    try:
        status, body = _post(addr, "/anomaly", {"measurements": telemetry_series})
        assert status == 200
        assert len(body["results"]) == len(telemetry_series)
        assert any(r["anomaly"] for r in body["results"])
    finally:
        server.shutdown()


def test_unknown_route(design_base):
    server, addr = _serve()
    try:
        status, _ = _post(addr, "/nope", {"target": {}, "candidates": design_base})
        assert status == 404
    finally:
        server.shutdown()
