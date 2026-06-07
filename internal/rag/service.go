package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/madragana/routefast-ee/internal/storage"
)

// Service is the RAG copilot entry point. Safe for concurrent use.
type Service struct {
	cfg   Config
	store *storage.Store
	embed Embedder
	gen   Generator
}

// Answer is the result of an operator question.
type Answer struct {
	Text    string   `json:"text"`
	Sources []string `json:"sources"`
}

// New constructs a Service over an existing storage pool.
func New(cfg Config, store *storage.Store) (*Service, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("rag: disabled (set RAG_ENABLED=true)")
	}
	if store == nil {
		return nil, fmt.Errorf("rag: nil store")
	}
	embed, err := newEmbedder(cfg)
	if err != nil {
		return nil, err
	}
	gen, err := newGenerator(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, store: store, embed: embed, gen: gen}, nil
}

// renderAudit builds the text that gets embedded and retrieved for one record.
func renderAudit(e storage.AuditEntry) string {
	node := e.NodeID
	if node == "" {
		node = "(unset)"
	}
	detail := strings.TrimSpace(string(e.Detail))
	if detail == "" {
		detail = "(no detail)"
	}
	return fmt.Sprintf("Event %s from node %s at %s. Detail: %s",
		e.EventType, node, e.LoggedAt.Format("2006-01-02 15:04:05Z"), detail)
}

// Reindex embeds audit records that are not yet in the vector store. It is
// incremental and safe to re-run: only unembedded rows are processed. Returns
// the number of records embedded.
func (s *Service) Reindex(ctx context.Context) (int, error) {
	total := 0
	for {
		rows, err := s.store.UnembeddedAudit(ctx, s.cfg.ReindexBatch)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			break
		}
		for _, e := range rows {
			content := renderAudit(e)
			vec, err := s.embed.Embed(ctx, content)
			if err != nil {
				return total, fmt.Errorf("rag: embed audit %s: %w", e.ID, err)
			}
			if len(vec) != s.cfg.EmbedDim {
				return total, fmt.Errorf("rag: embedding dimension %d does not match configured %d; align vector(N) in migration 004 and RAG_EMBED_DIM with your embedding model", len(vec), s.cfg.EmbedDim)
			}
			if err := s.store.UpsertAuditEmbedding(ctx, storage.AuditMatch{
				AuditID:    e.ID,
				CustomerID: e.CustomerID,
				EventType:  e.EventType,
				Content:    content,
				LoggedAt:   e.LoggedAt,
			}, vec); err != nil {
				return total, err
			}
			total++
		}
		if len(rows) < s.cfg.ReindexBatch {
			break
		}
	}
	return total, nil
}

// Ask answers an operator question grounded in a single customer's audit
// records. Retrieval is always scoped to customerID.
func (s *Service) Ask(ctx context.Context, customerID, question string) (Answer, error) {
	customerID = strings.TrimSpace(customerID)
	question = strings.TrimSpace(question)
	if customerID == "" {
		return Answer{}, fmt.Errorf("rag: customer_id required")
	}
	if question == "" {
		return Answer{}, fmt.Errorf("rag: empty question")
	}
	vec, err := s.embed.Embed(ctx, question)
	if err != nil {
		return Answer{}, fmt.Errorf("rag: embed question: %w", err)
	}
	matches, err := s.store.SearchAuditEmbeddings(ctx, customerID, vec, s.cfg.TopK)
	if err != nil {
		return Answer{}, err
	}
	if len(matches) == 0 {
		return Answer{Text: "No indexed audit records for this customer yet. Run a reindex first (POST /api/v1/reindex)."}, nil
	}
	contextBlock, sources := buildContext(matches)
	text, err := s.gen.Generate(ctx, askSystemPrompt, askUserPrompt(question, contextBlock))
	if err != nil {
		return Answer{}, fmt.Errorf("rag: generation: %w", err)
	}
	return Answer{Text: text, Sources: sources}, nil
}
