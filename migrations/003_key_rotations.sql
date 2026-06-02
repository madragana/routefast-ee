-- 003_key_rotations.sql
-- Indexing + integrity for the key-rotation history.

CREATE INDEX IF NOT EXISTS idx_keyrot_node_time ON key_rotations (node_id, rotated_at DESC);
