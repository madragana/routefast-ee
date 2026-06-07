package api

import (
	"context"
	"net/http"
	"time"
)

// handleAsk answers an operator question grounded in a customer's audit
// records. Read-only and advisory: it never issues credentials, changes
// policy, or takes any action. Sits behind mTLS + bearer token.
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if s.RAG == nil {
		writeError(w, http.StatusServiceUnavailable, "rag disabled (set RAG_ENABLED=true)")
		return
	}
	var req struct {
		CustomerID string `json:"customer_id"`
		Question   string `json:"question"`
	}
	if err := decode(r, &req); err != nil || req.CustomerID == "" || req.Question == "" {
		writeError(w, http.StatusBadRequest, "customer_id and question required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	ans, err := s.RAG.Ask(ctx, req.CustomerID, req.Question)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ans)
}

// handleReindex embeds any audit records not yet in the vector store. It is an
// administrative maintenance operation; restrict access accordingly.
func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	if s.RAG == nil {
		writeError(w, http.StatusServiceUnavailable, "rag disabled (set RAG_ENABLED=true)")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	n, err := s.RAG.Reindex(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"indexed": n})
}
