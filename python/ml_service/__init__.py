"""Transformer ML service.

Stateless compute service: feature preparation/scaling, project similarity
and anomaly detection over transformer data. Serves JSON over HTTP and is
called by the Go API (Phases 9-11). No database access: the API gathers the
data, this service computes.
"""
