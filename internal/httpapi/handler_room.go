package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/LessUp/aurora-signal/internal/config"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID              string         `json:"id"`
		MaxParticipants int            `json:"maxParticipants"`
		Metadata        map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, 2001, "invalid_body", err.Error())
		return
	}
	rm, err := s.rooms.CreateRoom(req.ID, req.MaxParticipants)
	if err != nil {
		writeError(w, http.StatusBadRequest, 2001, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": rm.ID, "maxParticipants": rm.MaxParticipants})
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	roomID, participants, ok := s.rooms.RoomInfo(id)
	if !ok {
		writeError(w, http.StatusNotFound, 2004, "room_not_found", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           roomID,
		"participants": participants,
	})
}

func (s *Server) handleJoinToken(w http.ResponseWriter, r *http.Request) {
	// constant-time admin key check to prevent timing attacks
	if s.cfg.Security.AdminKey != "" {
		provided := r.Header.Get("X-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.cfg.Security.AdminKey)) != 1 {
			writeError(w, http.StatusUnauthorized, 2002, "unauthorized", nil)
			return
		}
	}
	var req struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
		Role        string `json:"role"`
		TTLSeconds  int    `json:"ttlSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 2001, "invalid_body", err.Error())
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, 2001, "missing userId", nil)
		return
	}
	req.Role = config.NormalizeRole(req.Role)
	if req.Role == "" {
		writeError(w, http.StatusBadRequest, 2001, "invalid role", nil)
		return
	}
	req.TTLSeconds = config.ValidateJoinTokenTTL(req.TTLSeconds)
	roomID := chi.URLParam(r, "id")
	tok, err := s.auth.SignJoinToken(req.UserID, roomID, req.Role, time.Duration(req.TTLSeconds)*time.Second, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 3000, "sign token failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expiresIn": req.TTLSeconds})
}
