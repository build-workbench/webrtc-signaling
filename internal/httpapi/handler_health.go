package httpapi

import (
	"net/http"
)

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
