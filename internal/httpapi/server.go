package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"sync"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"singal/internal/auth"
	"singal/internal/config"
	"singal/internal/room"
	redispubsub "singal/internal/store/redis"
)

type Server struct {
	cfg      *config.Config
	log      *zap.Logger
	rooms    *room.Manager
	auth     *auth.JWT
	upgrader websocket.Upgrader
	httpSrv  *http.Server
	nodeID   string
	bus      *redispubsub.Bus
	mu       sync.Mutex
	roomSubs map[string]int
}

func NewServer(cfg *config.Config, log *zap.Logger, rooms *room.Manager, authJWT *auth.JWT) *Server {
	s := &Server{cfg: cfg, log: log, rooms: rooms, auth: authJWT}
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     s.checkOrigin,
	}
	mux := chi.NewRouter()
	mux.Get("/healthz", s.handleHealth)
	mux.Get("/readyz", s.handleReady)
	mux.Get("/metrics", promhttp.Handler().ServeHTTP)

	mux.Get("/api/v1/ice-servers", s.handleICEServers)
	mux.Post("/api/v1/rooms", s.handleCreateRoom)
	mux.Get("/api/v1/rooms/{id}", s.handleGetRoom)
	mux.Post("/api/v1/rooms/{id}/join-token", s.handleJoinToken)

	mux.Get("/ws/v1", s.handleWS)

	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/demo/*", http.StripPrefix("/demo/", fileServer))
	mux.Get("/demo", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, path.Join("/demo/", "index.html"), http.StatusFound)
	})

	s.httpSrv = &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	s.nodeID = uuid.NewString()
	s.roomSubs = make(map[string]int)
	if cfg.Redis.Enabled {
		bus, err := redispubsub.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, s.nodeID, s.log)
		if err == nil {
			s.bus = bus
		} else {
			s.log.Warn("redis disabled due to init failure", zap.Error(err))
		}
	}
	return s
}

func (s *Server) Start() error {
	s.log.Info("http server starting", zap.String("addr", s.cfg.Server.Addr))
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.bus != nil {
		_ = s.bus.Close()
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) checkOrigin(r *http.Request) bool {
	allowed := s.cfg.Server.AllowedOrigins
	if len(allowed) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, o := range allowed {
		if strings.EqualFold(strings.TrimSpace(o), origin) {
			return true
		}
	}
	return false
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.handleHealth(w, r)
}

func (s *Server) handleICEServers(w http.ResponseWriter, r *http.Request) {
	resp := []map[string]any{}
	for _, u := range s.cfg.Turn.STUN {
		resp = append(resp, map[string]any{"urls": []string{u}})
	}
	for _, t := range s.cfg.Turn.TURN {
		resp = append(resp, map[string]any{
			"urls":       t.URLs,
			"username":   t.Username,
			"credential": t.Credential,
			"ttl":        t.TTL,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
		MaxParticipants int `json:"maxParticipants"`
		Metadata map[string]any `json:"metadata"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rm, err := s.rooms.CreateRoom(req.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, 2001, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": rm.ID})
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rm, ok := s.rooms.GetRoom(id)
	if !ok {
		writeError(w, http.StatusNotFound, 2004, "room_not_found", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           rm.ID,
		"participants": len(rm.Participants),
	})
}

func (s *Server) handleJoinToken(w http.ResponseWriter, r *http.Request) {
	// simple admin key check
	if s.cfg.Security.AdminKey != "" {
		if r.Header.Get("X-Admin-Key") != s.cfg.Security.AdminKey {
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
	if req.TTLSeconds <= 0 {
		req.TTLSeconds = 900
	}
	roomID := chi.URLParam(r, "id")
	tok, err := s.auth.SignJoinToken(req.UserID, roomID, req.Role, time.Duration(req.TTLSeconds)*time.Second, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 3000, "sign token failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expiresIn": req.TTLSeconds})
}
