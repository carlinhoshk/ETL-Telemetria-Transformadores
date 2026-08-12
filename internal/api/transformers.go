package api

import (
	"net/http"

	"etl-telemetria-transformadores/internal/store"
)

// handleHealth reports liveness plus readiness checks (DB + ML service).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"service": "ok"}
	if err := s.deps.Store.Ping(r.Context()); err != nil {
		checks["database"] = "unavailable: " + err.Error()
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded", "checks": checks, "version": s.deps.Version,
		})
		return
	}
	checks["database"] = "ok"
	checks["ml"] = "ok"
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "checks": checks, "version": s.deps.Version,
	})
}

func (s *Server) handleListTransformers(w http.ResponseWriter, r *http.Request) {
	limit, offset := pageParams(r)
	total, err := s.deps.Store.CountTransformers(r.Context())
	if err != nil {
		s.logger.Error("count transformers", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to list transformers")
		return
	}
	rows, err := s.deps.Store.ListTransformers(r.Context(), limit, offset)
	if err != nil {
		s.logger.Error("list transformers", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to list transformers")
		return
	}
	w.Header().Set("X-Total-Count", itoa(total))
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": total})
}

func (s *Server) handleGetTransformer(w http.ResponseWriter, r *http.Request) {
	tr, err := s.deps.Store.GetTransformer(r.Context(), r.PathValue("id"))
	if err != nil {
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "not_found", "transformer not found")
			return
		}
		s.logger.Error("get transformer", requestIDAttr(r.Context()), "error", err)
		writeErr(w, http.StatusInternalServerError, "internal", "failed to get transformer")
		return
	}
	writeJSON(w, http.StatusOK, tr)
}
