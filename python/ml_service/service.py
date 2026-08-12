"""Minimal JSON HTTP service for the ML operations (Phases 9-11).

Routes:
  GET  /health          -> {"status": "ok"}
  POST /similar         -> {target: {...}, candidates: [...], top_k?: n}
                          => {results: [{transformer_id, score}]}
  POST /anomaly         -> {measurements: [...]}
                          => {results: [{transformer_id, timestamp, anomaly, score}]}

Uses the standard library only; no framework dependency.
"""
from __future__ import annotations

import argparse
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlparse

from .anomaly import detect_anomalies
from .features import fit_design_features
from .similarity import similarity_scores


class MLHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args: Any) -> None:  # quiet
        pass

    def _send(self, code: int, payload: Any) -> None:
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_body(self) -> dict:
        length = int(self.headers.get("Content-Length", 0) or 0)
        raw = self.rfile.read(length) if length else b"{}"
        try:
            body = json.loads(raw) if raw else {}
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid JSON body: {exc}") from exc
        if not isinstance(body, dict):
            raise ValueError("body must be a JSON object")
        return body

    def do_GET(self) -> None:  # noqa: N802 (http.server API)
        if urlparse(self.path).path == "/health":
            self._send(200, {"status": "ok", "service": "transformer-ml"})
        else:
            self._send(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        try:
            body = self._read_body()
            if path == "/similar":
                self._similar(body)
            elif path == "/anomaly":
                self._anomaly(body)
            else:
                self._send(404, {"error": f"unknown route {path}"})
        except (KeyError, TypeError, ValueError) as exc:
            self._send(400, {"error": str(exc)})

    def _similar(self, body: dict) -> None:
        target = body["target"]
        candidates = body["candidates"]
        top_k = int(body.get("top_k", len(candidates)))
        model = fit_design_features(candidates)
        scores = similarity_scores(target, candidates, model)
        if top_k <= 0:
            top_k = len(scores)
        results = [
            {"transformer_id": rid, "score": score}
            for rid, score in scores[:top_k]
        ]
        self._send(200, {"results": results, "features": list(model.columns)})

    def _anomaly(self, body: dict) -> None:
        measurements = body["measurements"]
        contamination = float(body.get("contamination", 0.1))
        results = detect_anomalies(measurements, contamination=contamination)
        self._send(200, {"results": results})


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Transformer ML service")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8081)
    args = parser.parse_args(argv)

    server = ThreadingHTTPServer((args.host, args.port), MLHandler)
    print(f"ml service listening on {args.host}:{args.port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
