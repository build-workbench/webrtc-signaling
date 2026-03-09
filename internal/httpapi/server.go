package httpapi

import (
	"context"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/LessUp/aurora-signal/internal/auth"
	"github.com/LessUp/aurora-signal/internal/config"
	"github.com/LessUp/aurora-signal/internal/room"
	redispubsub "github.com/LessUp/aurora-signal/internal/store/redis"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
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

	// activeConns tracks WebSocket connections for graceful shutdown.
	connsMu     sync.Mutex
	activeConns map[*websocket.Conn]struct{}
}

func NewServer(cfg *config.Config, log *zap.Logger, rooms *room.Manager, authJWT *auth.JWT) *Server {
	s := &Server{cfg: cfg, log: log, rooms: rooms, auth: authJWT, activeConns: make(map[*websocket.Conn]struct{})}
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
	// Close all active WebSocket connections so their read loops exit cleanly.
	s.connsMu.Lock()
	for c := range s.activeConns {
		_ = c.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(2*time.Second),
		)
		_ = c.Close()
	}
	s.connsMu.Unlock()

	if s.bus != nil {
		_ = s.bus.Close()
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) trackConn(c *websocket.Conn) {
	s.connsMu.Lock()
	s.activeConns[c] = struct{}{}
	s.connsMu.Unlock()
}

func (s *Server) untrackConn(c *websocket.Conn) {
	s.connsMu.Lock()
	delete(s.activeConns, c)
	s.connsMu.Unlock()
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
