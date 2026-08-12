package api

import (
	"net/http"
	"strconv"
	"time"
)

const (
	defaultPageLimit = 100
	maxPageLimit     = 1000
)

func itoa(n int) string { return strconv.Itoa(n) }

// pageParams extracts limit/offset with sane defaults and caps.
func pageParams(r *http.Request) (limit, offset int) {
	limit = defaultPageLimit
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			offset = n
		}
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	return limit, offset
}

// timeParam parses an optional RFC3339 query parameter (zero value if empty).
func timeParam(r *http.Request, key string) time.Time {
	v := r.URL.Query().Get(key)
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}
