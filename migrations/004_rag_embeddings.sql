-- 004_rag_embeddings.sql
-- Vector store for the advisory RAG copilot. Embeds the append-only audit log
-- so operators can ask grounded, cited questions about a customer's fleet.
--
-- Requires YugabyteDB / PostgreSQL with the pgvector extension available.
-- The embedding dimension below (768) matches the default local model
-- (nomic-embed-text). If you switch RAG_EMBED_MODEL to a model with a
-- different dimension (e.g. OpenAI text-embedding-3-small is 1536), change
-- vector(768) to match and set RAG_EMBED_DIM accordingly.

CREATE EXTENSION IF NOT EXISTS "vector";

CREATE TABLE IF NOT EXISTS audit_embeddings (
    audit_id    UUID PRIMARY KEY REFERENCES audit_logs (id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    event_type  TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL,
    embedding   vector(768) NOT NULL,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    embedded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Retrieval is always scoped to a single customer.
CREATE INDEX IF NOT EXISTS idx_audit_emb_customer ON audit_embeddings (customer_id);

-- An approximate-nearest-neighbour index (ivfflat/hnsw) can be added once
-- verified against your YugabyteDB version. Exact search via ORDER BY <=>
-- works without one and is fine for typical audit volumes.
