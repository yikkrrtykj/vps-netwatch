package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
	"github.com/yikkrrtykj/vps-netwatch/internal/store"
)

type Server struct {
	cfg   config.Config
	store *store.Store
	mux   *http.ServeMux
}

func New(cfg config.Config, store *store.Store) *Server {
	s := &Server{
		cfg:   cfg,
		store: store,
		mux:   http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withAuth(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/connections", s.connections)
	s.mux.HandleFunc("GET /api/egress", s.state("egress", model.EgressResult{}))
	s.mux.HandleFunc("GET /api/latency", s.state("latency", []model.ProbeResult{}))
	s.mux.HandleFunc("GET /api/errors", s.state("collector_errors", []model.CollectorError{}))
	s.mux.HandleFunc("GET /api/vps/nodes", s.vpsNodes)
	s.mux.HandleFunc("POST /api/collector/v1/push", s.collectorPush)
	s.mux.Handle("/", http.FileServer(http.Dir("web/dist")))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) connections(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	connections, err := s.store.LatestConnections(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, connections)
}

func (s *Server) state(key string, empty any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value := cloneEmpty(empty)
		ok, err := s.store.LoadState(r.Context(), key, value)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, empty)
			return
		}
		writeJSON(w, http.StatusOK, value)
	}
}

func (s *Server) vpsNodes(w http.ResponseWriter, r *http.Request) {
	var nodes []model.VPSNodeStatus
	ok, err := s.store.LoadState(r.Context(), "vps_nodes", &nodes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ok {
		writeJSON(w, http.StatusOK, nodes)
		return
	}
	fallback := make([]model.VPSNodeStatus, 0, len(s.cfg.VPSNodes))
	for _, node := range s.cfg.VPSNodes {
		fallback = append(fallback, model.VPSNodeStatus{
			ID:        node.ID,
			Name:      node.Name,
			PublicIP:  node.PublicIP,
			Labels:    node.Labels,
			UpdatedAt: time.Now().UTC(),
		})
	}
	writeJSON(w, http.StatusOK, fallback)
}

func (s *Server) collectorPush(w http.ResponseWriter, r *http.Request) {
	var push model.CollectorPush
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if push.CollectorID == "" {
		writeErrorMessage(w, http.StatusBadRequest, "collector_id is required")
		return
	}
	if push.Timestamp.IsZero() {
		push.Timestamp = time.Now().UTC()
	}
	if err := s.store.SaveCollectorPush(r.Context(), push); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok": true,
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || s.cfg.Auth.Token == "" || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" {
			got = r.Header.Get("X-Netwatch-Token")
		}
		if got == "" {
			got = r.URL.Query().Get("token")
		}
		if got != s.cfg.Auth.Token {
			writeErrorMessage(w, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorMessage(w, status, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}

func cloneEmpty(value any) any {
	switch value.(type) {
	case model.EgressResult:
		return &model.EgressResult{}
	case []model.ProbeResult:
		return &[]model.ProbeResult{}
	case []model.CollectorError:
		return &[]model.CollectorError{}
	default:
		return value
	}
}
