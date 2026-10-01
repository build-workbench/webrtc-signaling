package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/build-workbench/webrtc-signaling/internal/config"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID              string `json:"id"`
		MaxParticipants int    `json:"maxParticipants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeErrorWithMetrics(w, http.StatusBadRequest, 2001, "invalid_body", err.Error(), s.metrics)
		return
	}
	rm, err := s.rooms.CreateRoom(req.ID, req.MaxParticipants)
	if err != nil {
		writeErrorWithMetrics(w, http.StatusBadRequest, 2001, err.Error(), nil, s.metrics)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": rm.ID, "maxParticipants": rm.MaxParticipants})
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	roomID, participants, ok := s.rooms.RoomInfo(id)
	if !ok {
		writeErrorWithMetrics(w, http.StatusNotFound, 2004, "room_not_found", nil, s.metrics)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           roomID,
		"participants": participants,
	})
}

func (s *Server) handleJoinToken(w http.ResponseWriter, r *http.Request) {
	// AdminKey 是签发 join-token 的唯一门槛：未配置时拒绝签发，避免默认部署
	// 下任何人自签任意房间/任意角色的令牌（fail-closed）。
	if s.cfg.Security.AdminKey == "" {
		writeErrorWithMetrics(w, http.StatusServiceUnavailable, 2002, "admin_key_not_configured", "SIGNAL_ADMIN_KEY is not set; token issuance disabled", s.metrics)
		return
	}
	// constant-time admin key check to prevent timing attacks
	provided := r.Header.Get("X-Admin-Key")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(s.cfg.Security.AdminKey)) != 1 {
		writeErrorWithMetrics(w, http.StatusUnauthorized, 2002, "unauthorized", nil, s.metrics)
		return
	}
	var req struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
		Role        string `json:"role"`
		TTLSeconds  int    `json:"ttlSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorWithMetrics(w, http.StatusBadRequest, 2001, "invalid_body", err.Error(), s.metrics)
		return
	}
	if req.UserID == "" {
		writeErrorWithMetrics(w, http.StatusBadRequest, 2001, "missing userId", nil, s.metrics)
		return
	}
	normalizedRole := s.policy.Normalize(req.Role)
	if normalizedRole == "" {
		writeErrorWithMetrics(w, http.StatusBadRequest, 2001, "invalid role", nil, s.metrics)
		return
	}
	req.TTLSeconds = config.ValidateJoinTokenTTL(req.TTLSeconds)
	roomID := chi.URLParam(r, "id")
	tok, err := s.auth.SignJoinToken(req.UserID, roomID, string(normalizedRole), time.Duration(req.TTLSeconds)*time.Second, req.DisplayName)
	if err != nil {
		writeErrorWithMetrics(w, http.StatusInternalServerError, 3000, "sign token failed", err.Error(), s.metrics)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expiresIn": req.TTLSeconds})
}
