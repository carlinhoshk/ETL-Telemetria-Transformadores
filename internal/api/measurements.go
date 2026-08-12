package api

import (
	"encoding/json"
	"net/http"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/store"
)

// handleTelemetry returns measurements in a time window (RFC3339 from/to).
func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.deps.Store.GetTransformer(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "transformer not found")
		return
	}
	limit, offset := pageParams(r)
	from := timeParam(r, "from")
	to := timeParam(r, "to")

	total, err := s.deps.Store.CountTelemetry(r.Context(), id, from, to)
	if err != nil {
		s.logger.Error("count telemetry", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to query telemetry")
		return
	}
	rows, err := s.deps.Store.ListTelemetry(r.Context(), id, from, to, limit, offset)
	if err != nil {
		s.logger.Error("list telemetry", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to query telemetry")
		return
	}
	w.Header().Set("X-Total-Count", itoa(total))
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": total})
}

// handleEvents returns a transformer's events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.deps.Store.GetTransformer(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "transformer not found")
		return
	}
	limit, offset := pageParams(r)
	rows, err := s.deps.Store.ListEvents(r.Context(), id, limit, offset)
	if err != nil {
		s.logger.Error("list events", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to query events")
		return
	}
	if rows == nil {
		rows = []store.Event{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows})
}

// handleStatistics returns aggregated measurement statistics.
func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.deps.Store.GetTransformer(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "transformer not found")
		return
	}
	stats, err := s.deps.Store.TransformerStatistics(r.Context(), id)
	if err != nil {
		s.logger.Error("statistics", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to compute statistics")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleSimilar calls the ML service for similar transformers.
func (s *Server) handleSimilar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	target, err := s.deps.Store.GetTransformer(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "not_found", "transformer not found")
			return
		}
		s.logger.Error("similar target", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to load transformer")
		return
	}

	candidates, err := s.deps.Store.ListTransformers(r.Context(), 1000, 0)
	if err != nil {
		s.logger.Error("similar candidates", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to load candidates")
		return
	}
	// Exclude the target itself from its own candidate list.
	filtered := candidates[:0]
	for _, c := range candidates {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}

	matches, err := s.deps.ML.Similar(r.Context(), target, filtered, 5)
	if err != nil {
		s.logger.Error("ml similar", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusServiceUnavailable, "ml_unavailable", "similarity service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transformer_id": id,
		"matches":        matches,
	})
}

// handleCreateTransformer registers a new design record.
func (s *Server) handleCreateTransformer(w http.ResponseWriter, r *http.Request) {
	var tr domain.Transformer
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&tr); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := tr.Validate(); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if err := s.deps.Store.InsertTransformer(r.Context(), tr); err != nil {
		if err == store.ErrConflict {
			writeErr(w, http.StatusConflict, "conflict", "transformer already exists")
			return
		}
		s.logger.Error("create transformer", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to create transformer")
		return
	}
	writeJSON(w, http.StatusCreated, tr)
}
