-- 002_audit_indexes.sql
-- Performance indexes for the most common lookups.

CREATE INDEX IF NOT EXISTS idx_nodes_customer       ON nodes (customer_id);
CREATE INDEX IF NOT EXISTS idx_tokens_node          ON tokens (node_id);
CREATE INDEX IF NOT EXISTS idx_tokens_expires       ON tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_audit_customer_time  ON audit_logs (customer_id, logged_at);
CREATE INDEX IF NOT EXISTS idx_audit_node           ON audit_logs (node_id);
