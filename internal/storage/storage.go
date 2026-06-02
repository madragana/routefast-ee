// Package storage is the data-access layer for the credential server. It talks
// to YugabyteDB over the PostgreSQL wire protocol via the pgx driver. All
// queries are parameterised; the audit log is append-only.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madragana/routefast-ee/internal/crypto"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Store wraps a YugabyteDB connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// Node is a registered RouteFast CE node.
type Node struct {
	ID         string
	CustomerID string
	PublicKey  string
	CreatedAt  time.Time
}

// Token is a 7-day bearer token issued to a node.
type Token struct {
	Token     string
	NodeID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Revoked   bool
}

// Policy is the policy bundle distributed to a customer's nodes.
type Policy struct {
	CustomerID string
	Document   []byte // JSON: quorum rules, mitigation templates, evidence schemas
	Checksum   string
	UpdatedAt  time.Time
}

// AuditEntry is one append-only decision/audit record.
type AuditEntry struct {
	ID         string
	CustomerID string
	NodeID     string
	EventType  string
	Detail     []byte // JSON
	LoggedAt   time.Time
}

// New opens a connection pool to YugabyteDB.
func New(ctx context.Context, databaseURL string, maxConns int32) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Health verifies the database is reachable.
func (s *Store) Health(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// RegisterNode generates an ed25519 keypair for a new node, persists the public
// key, and returns the created node plus the private key (returned to caller
// once and never stored server-side beyond the public half).
func (s *Store) RegisterNode(ctx context.Context, customerID string) (*Node, *crypto.Keypair, error) {
	kp, err := crypto.GenerateKeypair()
	if err != nil {
		return nil, nil, err
	}
	var (
		id        string
		createdAt time.Time
	)
	err = s.pool.QueryRow(ctx,
		`INSERT INTO nodes (customer_id, public_key)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		customerID, kp.PublicKey,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, nil, fmt.Errorf("insert node: %w", err)
	}
	return &Node{ID: id, CustomerID: customerID, PublicKey: kp.PublicKey, CreatedAt: createdAt}, kp, nil
}

// GenerateToken issues a fresh bearer token for a node valid for ttl.
func (s *Store) GenerateToken(ctx context.Context, nodeID string, ttl time.Duration) (*Token, error) {
	raw, err := crypto.GenerateToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	exp := now.Add(ttl)
	_, err = s.pool.Exec(ctx,
		`INSERT INTO tokens (token, node_id, issued_at, expires_at, revoked)
		 VALUES ($1, $2, $3, $4, false)`,
		raw, nodeID, now, exp,
	)
	if err != nil {
		return nil, fmt.Errorf("insert token: %w", err)
	}
	return &Token{Token: raw, NodeID: nodeID, IssuedAt: now, ExpiresAt: exp}, nil
}

// RefreshToken revokes the presented token and issues a new one for the same
// node, provided the old token is valid and unexpired.
func (s *Store) RefreshToken(ctx context.Context, oldToken string, ttl time.Duration) (*Token, error) {
	var (
		nodeID  string
		expires time.Time
		revoked bool
	)
	err := s.pool.QueryRow(ctx,
		`SELECT node_id, expires_at, revoked FROM tokens WHERE token = $1`,
		oldToken,
	).Scan(&nodeID, &expires, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup token: %w", err)
	}
	if revoked || time.Now().UTC().After(expires) {
		return nil, errors.New("token revoked or expired")
	}
	if _, err := s.pool.Exec(ctx, `UPDATE tokens SET revoked = true WHERE token = $1`, oldToken); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}
	return s.GenerateToken(ctx, nodeID, ttl)
}

// ValidateToken returns the node ID for a live (unrevoked, unexpired) token.
func (s *Store) ValidateToken(ctx context.Context, token string) (nodeID string, err error) {
	var (
		expires time.Time
		revoked bool
	)
	err = s.pool.QueryRow(ctx,
		`SELECT node_id, expires_at, revoked FROM tokens WHERE token = $1`,
		token,
	).Scan(&nodeID, &expires, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if revoked || time.Now().UTC().After(expires) {
		return "", errors.New("token revoked or expired")
	}
	return nodeID, nil
}

// RotateKey generates a new ed25519 keypair for a node, records the rotation in
// the append-only key_rotations table, and updates the node's active key.
func (s *Store) RotateKey(ctx context.Context, nodeID string) (*crypto.Keypair, error) {
	kp, err := crypto.GenerateKeypair()
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO key_rotations (node_id, public_key, rotated_at)
		 VALUES ($1, $2, now())`,
		nodeID, kp.PublicKey,
	); err != nil {
		return nil, fmt.Errorf("record rotation: %w", err)
	}
	tag, err := tx.Exec(ctx,
		`UPDATE nodes SET public_key = $2 WHERE id = $1`,
		nodeID, kp.PublicKey,
	)
	if err != nil {
		return nil, fmt.Errorf("update node key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return kp, nil
}

// GetPolicies returns the active policy bundle for a customer.
func (s *Store) GetPolicies(ctx context.Context, customerID string) (*Policy, error) {
	p := &Policy{CustomerID: customerID}
	err := s.pool.QueryRow(ctx,
		`SELECT document, checksum, updated_at FROM policies WHERE customer_id = $1`,
		customerID,
	).Scan(&p.Document, &p.Checksum, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get policies: %w", err)
	}
	return p, nil
}

// LogDecision appends an audit record. The audit_logs table is append-only.
func (s *Store) LogDecision(ctx context.Context, e AuditEntry) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO audit_logs (customer_id, node_id, event_type, detail, logged_at)
		 VALUES ($1, $2, $3, $4, now())
		 RETURNING id`,
		e.CustomerID, e.NodeID, e.EventType, e.Detail,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("append audit: %w", err)
	}
	return id, nil
}

// GetAuditTrail returns audit records for a customer within a date range.
func (s *Store) GetAuditTrail(ctx context.Context, customerID string, start, end time.Time) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, customer_id, node_id, event_type, detail, logged_at
		 FROM audit_logs
		 WHERE customer_id = $1 AND logged_at >= $2 AND logged_at < $3
		 ORDER BY logged_at ASC`,
		customerID, start, end,
	)
	if err != nil {
		return nil, fmt.Errorf("query audit trail: %w", err)
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
