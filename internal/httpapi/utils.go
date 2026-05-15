// Package httpapi provides HTTP and WebSocket handling for the signaling server.
//
// This file contains convenience functions for JSON handling. These are not
// architectural abstractions but simple wrappers to reduce boilerplate.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/LessUp/aurora-signal/internal/observability"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErrorWithMetrics writes an error response and records metrics.
func writeErrorWithMetrics(w http.ResponseWriter, httpStatus, code int, message string, details any, metrics observability.Metrics) {
	metrics.IncError(code)
	writeJSON(w, httpStatus, map[string]any{
		"type":    "error",
		"payload": map[string]any{"code": code, "message": message, "details": details},
	})
}

// mustJSON marshals v to JSON. It panics on error, so only use for values
// that are guaranteed to marshal correctly (e.g., simple maps and structs).
func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
