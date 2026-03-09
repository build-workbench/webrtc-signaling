package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/LessUp/aurora-signal/internal/observability"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, httpStatus, code int, message string, details any) {
	observability.ErrorsTotal.WithLabelValues(strconv.Itoa(code)).Inc()
	writeJSON(w, httpStatus, map[string]any{
		"type":    "error",
		"payload": map[string]any{"code": code, "message": message, "details": details},
	})
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
