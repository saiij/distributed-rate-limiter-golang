package api

import (
	"encoding/json"
	"net/http"

	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

type Server struct {
	mux         *http.ServeMux
	rateLimiter ratelimiter.RateLimiter
}

func NewServer(rateLimiter ratelimiter.RateLimiter) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		rateLimiter: rateLimiter,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /check", s.handleCheck)
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// POST /check
type checkRequest struct {
	Key string `json:"key"`
}

type checkResponse struct {
	CanAccess      bool   `json:"can_access"`
	RequestRemain  int    `json:"request_remain"`
	RetryIn        string `json:"retry_in"`
	ResetRequestAt string `json:"reset_request_at"`
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	result, err := s.rateLimiter.Allow(r.Context(), req.Key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	if !result.CanAccess {
		status = http.StatusTooManyRequests
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(checkResponse{
		CanAccess:      result.CanAccess,
		RequestRemain:  result.RequestRemain,
		RetryIn:        result.RetryIn.String(),
		ResetRequestAt: result.ResetRequestAt.UTC().Format(http.TimeFormat),
	})
}

// GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
