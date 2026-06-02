-- 001_initial_schema.sql
-- RouteFast Enterprise Edition — initial schema for the lipd-server.
-- Target: YugabyteDB 3.0+ (PostgreSQL wire compatible).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Customers (tenants) holding subscriptions.
CREATE TABLE IF NOT EXISTS customers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    tier        TEXT NOT NULL DEFAULT 'standard',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Registered RouteFast CE nodes, each with its active ed25519 public key.
CREATE TABLE IF NOT EXISTS nodes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    public_key  TEXT NOT NULL,          -- base64 ed25519 public key
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Bearer tokens (7-day default), refreshable. Old tokens are revoked, not deleted.
CREATE TABLE IF NOT EXISTS tokens (
    token       TEXT PRIMARY KEY,
    node_id     UUID NOT NULL,
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT false
);

-- Policy bundles distributed to a customer's nodes.
CREATE TABLE IF NOT EXISTS policies (
    customer_id UUID PRIMARY KEY,
    document    JSONB NOT NULL,         -- quorum rules, mitigation templates, evidence schemas
    checksum    TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only audit/decision log (SOC 2 / HIPAA evidence).
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    node_id     UUID,
    event_type  TEXT NOT NULL,
    detail      JSONB,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- History of ed25519 key rotations (append-only).
CREATE TABLE IF NOT EXISTS key_rotations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id     UUID NOT NULL,
    public_key  TEXT NOT NULL,
    rotated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
