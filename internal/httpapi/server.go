package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"runtime/debug"
	"sync"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"signal/internal/auth"
	"signal/internal/config"
	"signal/internal/room"
	redispubsub "signal/internal/store/redis"
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
	mux.Use(s.recoveryMiddleware)
	mux.Use(s.requestIDMiddleware)
	mux.Use(s.corsMiddleware)
	mux.Use(securityHeadersMiddleware)
	mux.Use(s.accessLogMiddleware)

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
	if s.bus != nil {
		if err := s.bus.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("redis unreachable"))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) buildICEServers() []map[string]any {
	resp := make([]map[string]any, 0, len(s.cfg.Turn.STUN)+len(s.cfg.Turn.TURN))
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
	return resp
}

func (s *Server) handleICEServers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildICEServers())
}

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
	rm, err := s.rooms.CreateRoom(req.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, 2001, err.Error(), nil)
		return
	}
	if req.MaxParticipants > 0 {
		rm.MaxParticipants = req.MaxParticipants
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": rm.ID, "maxParticipants": rm.MaxParticipants})
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

// --- Middleware ---

type ctxKey string

const reqIDKey ctxKey = "reqID"

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				s.log.Error("panic recovered",
					zap.Any("error", rv),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", r.URL.Path),
				)
				writeError(w, http.StatusInternalServerError, 3000, "internal_error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), reqIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.checkOrigin(r) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (s *Server) accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// skip noisy endpoints
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("duration", time.Since(start)),
			zap.String("reqID", requestID(r)),
		)
	})
}

func requestID(r *http.Request) string {
	if id, ok := r.Context().Value(reqIDKey).(string); ok {
		return id
	}
	return ""
}
