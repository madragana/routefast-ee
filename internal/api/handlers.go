// Package api implements the REST/JSON handlers for the credential server.
// Every endpoint (except /health) sits behind mTLS at the transport layer;
// node-scoped endpoints additionally require a valid bearer token.
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/madragana/routefast-ee/internal/config"
	"github.com/madragana/routefast-ee/internal/storage"
)

// Server bundles the dependencies the handlers need.
type Server struct {
	Store *storage.Store
	Cfg   *config.Config
}

// Routes returns the configured mux with all 7 endpoints registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/token/refresh", s.handleTokenRefresh)
	mux.HandleFunc("POST /api/v1/key/rotate", s.requireToken(s.handleKeyRotate))
	mux.HandleFunc("GET /api/v1/policies", s.requireToken(s.handlePolicies))
	mux.HandleFunc("POST /api/v1/audit/log", s.requireToken(s.handleAuditLog))
	mux.HandleFunc("GET /api/v1/audit/trail", s.requireToken(s.handleAuditTrail))
	return logging(mux)
}

// ---- 1. Health -------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := s.Store.Health(r.Context()); err != nil {
		dbStatus = "error"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "healthy",
		"uptime":   time.Now().Unix(),
		"database": dbStatus,
	})
}

// ---- 2. Register -----------------------------------------------------------

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID    string `json:"customer_id"`
		InitialSecret string `json:"initial_secret"`
	}
	if err := decode(r, &req); err != nil || req.CustomerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id required")
		return
	}

	node, kp, err := s.Store.RegisterNode(r.Context(), req.CustomerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "register failed")
		return
	}
	tok, err := s.Store.GenerateToken(r.Context(), node.ID, s.Cfg.TokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	policies, _ := s.Store.GetPolicies(r.Context(), req.CustomerID)

	resp := map[string]any{
		"node_id":     node.ID,
		"public_key":  kp.PublicKey,
		"private_key": kp.PrivateKey, // returned once, never stored server-side
		"token":       tok.Token,
		"expires_at":  tok.ExpiresAt,
	}
	if policies != nil {
		resp["policies_checksum"] = policies.Checksum
	}
	writeJSON(w, http.StatusCreated, resp)
}

// ---- 3. Token refresh ------------------------------------------------------

func (s *Server) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	old := bearer(r)
	if old == "" {
		writeError(w, http.StatusUnauthorized, "bearer token required")
		return
	}
	tok, err := s.Store.RefreshToken(r.Context(), old, s.Cfg.TokenTTL)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      tok.Token,
		"expires_at": tok.ExpiresAt,
	})
}

// ---- 4. Key rotation -------------------------------------------------------

func (s *Server) handleKeyRotate(w http.ResponseWriter, r *http.Request) {
	nodeID := nodeFromContext(r)
	kp, err := s.Store.RotateKey(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rotation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"public_key":  kp.PublicKey,
		"private_key": kp.PrivateKey,
		"rotated_at":  time.Now().UTC(),
	})
}

// ---- 5. Get policies -------------------------------------------------------

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id required")
		return
	}
	p, err := s.Store.GetPolicies(r.Context(), customerID)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no policies for customer")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "policy fetch failed")
		return
	}
	w.Header().Set("X-Policy-Checksum", p.Checksum)
	writeJSON(w, http.StatusOK, json.RawMessage(p.Document))
}

// ---- 6. Audit log (append) -------------------------------------------------

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string          `json:"customer_id"`
		EventType  string          `json:"event_type"`
		Detail     json.RawMessage `json:"detail"`
	}
	if err := decode(r, &req); err != nil || req.CustomerID == "" || req.EventType == "" {
		writeError(w, http.StatusBadRequest, "customer_id and event_type required")
		return
	}
	id, err := s.Store.LogDecision(r.Context(), storage.AuditEntry{
		CustomerID: req.CustomerID,
		NodeID:     nodeFromContext(r),
		EventType:  req.EventType,
		Detail:     req.Detail,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit append failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// ---- 7. Audit trail (query) ------------------------------------------------

func (s *Server) handleAuditTrail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	customerID := q.Get("customer_id")
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id required")
		return
	}
	start := parseDate(q.Get("start_date"), time.Now().AddDate(0, -1, 0))
	end := parseDate(q.Get("end_date"), time.Now()).Add(24 * time.Hour)

	entries, err := s.Store.GetAuditTrail(r.Context(), customerID, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer_id": customerID,
		"count":       len(entries),
		"entries":     entries,
	})
}

// ---- middleware & helpers --------------------------------------------------

type ctxKey string

const nodeIDKey ctxKey = "node_id"

// requireToken validates the bearer token and stashes the node ID in context.
func (s *Server) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		if tok == "" {
			writeError(w, http.StatusUnauthorized, "bearer token required")
			return
		}
		nodeID, err := s.Store.ValidateToken(r.Context(), tok)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next(w, r.WithContext(withNode(r, nodeID)))
	}
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseDate(s string, fallback time.Time) time.Time {
	if s == "" {
		return fallback
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fallback
	}
	return t
}

func checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
