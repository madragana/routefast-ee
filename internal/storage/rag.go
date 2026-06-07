// RAG vector-store access. Keeps all SQL in the storage layer, consistent with
// the rest of this package. Embeddings are stored in audit_embeddings (see
// migrations/004) and searched with pgvector's cosine distance operator.
package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AuditMatch is an audit record retrieved for RAG, with its similarity to a
// query (1.0 = identical, 0.0 = orthogonal). On the write path Similarity is
// unused.
type AuditMatch struct {
	AuditID    string
	CustomerID string
	EventType  string
	Content    string
	LoggedAt   time.Time
	Similarity float64
}

// vectorLiteral renders a float32 slice as a pgvector text literal "[1,2,3]".
// Passed as a bind parameter and cast with ::vector in SQL.
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v)*8 + 2)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// UnembeddedAudit returns audit rows that do not yet have an embedding, oldest
// first, up to limit. Used by the RAG reindexer for incremental embedding.
func (s *Store) UnembeddedAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT a.id::text, a.customer_id::text, COALESCE(a.node_id::text, ''),
		        a.event_type, a.detail, a.logged_at
		 FROM audit_logs a
		 LEFT JOIN audit_embeddings e ON e.audit_id = a.id
		 WHERE e.audit_id IS NULL
		 ORDER BY a.logged_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query unembedded audit: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.NodeID, &e.EventType, &e.Detail, &e.LoggedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertAuditEmbedding stores (or replaces) the embedding for one audit record.
func (s *Store) UpsertAuditEmbedding(ctx context.Context, m AuditMatch, embedding []float32) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_embeddings (audit_id, customer_id, event_type, content, logged_at, embedding)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::vector)
		 ON CONFLICT (audit_id) DO UPDATE
		   SET content     = EXCLUDED.content,
		       event_type  = EXCLUDED.event_type,
		       embedding   = EXCLUDED.embedding,
		       embedded_at = now()`,
		m.AuditID, m.CustomerID, m.EventType, m.Content, m.LoggedAt, vectorLiteral(embedding),
	)
	if err != nil {
		return fmt.Errorf("upsert audit embedding: %w", err)
	}
	return nil
}

// SearchAuditEmbeddings returns the k audit records for a customer whose
// embeddings are closest to the query embedding (cosine similarity).
func (s *Store) SearchAuditEmbeddings(ctx context.Context, customerID string, query []float32, k int) ([]AuditMatch, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT audit_id::text, customer_id::text, event_type, content, logged_at,
		        1 - (embedding <=> $1::vector) AS similarity
		 FROM audit_embeddings
		 WHERE customer_id = $2::uuid
		 ORDER BY embedding <=> $1::vector
		 LIMIT $3`,
		vectorLiteral(query), customerID, k,
	)
	if err != nil {
		return nil, fmt.Errorf("search audit embeddings: %w", err)
	}
	defer rows.Close()

	var out []AuditMatch
	for rows.Next() {
		var m AuditMatch
		if err := rows.Scan(&m.AuditID, &m.CustomerID, &m.EventType, &m.Content, &m.LoggedAt, &m.Similarity); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
