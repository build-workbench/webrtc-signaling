package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"singal/internal/observability"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, httpStatus, code int, message string, details interface{}) {
	observability.ErrorsTotal.WithLabelValues(strconv.Itoa(code)).Inc()
	writeJSON(w, httpStatus, map[string]interface{}{
		"type":    "error",
		"payload": map[string]interface{}{"code": code, "message": message, "details": details},
	})
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
