# RouteFast EE RAG copilot (`internal/rag`)

An advisory Retrieval-Augmented Generation copilot for the EE control plane. It
indexes the append-only audit log into a `pgvector` store and answers
natural-language questions about a customer's fleet, grounded in that
customer's own audit records, with citations.

It is strictly read-only and advisory. It never issues credentials, changes
policy, mutates the audit log, or takes any action. Retrieval is always scoped
to a single `customer_id`. It is off by default (`RAG_ENABLED`).

It adds no new Go dependencies: vectors are stored and searched with `pgvector`
on the existing YugabyteDB pool, and the embedding/LLM clients are plain HTTP.

## Prerequisites

- YugabyteDB / PostgreSQL with the `pgvector` extension available.
- Apply the migration: `migrations/004_rag_embeddings.sql` (creates the
  `vector` extension and the `audit_embeddings` table).
- An embedding and a generation backend. Local default is Ollama:
  `ollama pull nomic-embed-text` and `ollama pull llama3.1`.

The embedding dimension in the migration is `vector(768)`, matching
`nomic-embed-text`. If you switch to a model with a different dimension (for
example OpenAI `text-embedding-3-small` is 1536), change `vector(768)` in the
migration and set `RAG_EMBED_DIM` to match. The reindexer fails fast with a
clear error if the model's dimension does not match the configured one.

## Enabling

```bash
export RAG_ENABLED=true
# start lipd-server as usual, then build the index once:
curl -sk -X POST https://<server>/api/v1/reindex -H "Authorization: Bearer <token>"
curl -sk -X POST https://<server>/api/v1/ask \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"customer_id":"<uuid>","question":"why did any node blackhole a prefix yesterday?"}'
```

## Endpoints

Both sit behind the existing mTLS and bearer-token middleware.

- `POST /api/v1/ask` — body `{"customer_id": "...", "question": "..."}` →
  `{"text": "...", "sources": [...]}`. Retrieval is scoped to `customer_id`.
- `POST /api/v1/reindex` — embeds audit records not yet in the store
  (incremental, safe to re-run) → `{"indexed": N}`. Administrative maintenance;
  restrict access accordingly.

Both return `503` when RAG is disabled.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `RAG_ENABLED` | `false` | Master switch |
| `RAG_TOP_K` | `6` | Audit records retrieved per question |
| `RAG_REINDEX_BATCH` | `128` | Rows embedded per batch |
| `RAG_EMBED_BACKEND` | `ollama` | `ollama` or `openai` |
| `RAG_EMBED_MODEL` | backend-specific | e.g. `nomic-embed-text`, `text-embedding-3-small` |
| `RAG_EMBED_BASE_URL` | backend default | Override embedding endpoint |
| `RAG_EMBED_API_KEY` | — | For `openai` embeddings |
| `RAG_EMBED_DIM` | `768` | Must match `vector(N)` in migration 004 |
| `RAG_LLM_BACKEND` | `ollama` | `ollama`, `openai`, or `anthropic` |
| `RAG_LLM_MODEL` | backend-specific | e.g. `llama3.1`, `gpt-4o-mini`, `claude-3-5-haiku-latest` |
| `RAG_LLM_BASE_URL` | backend default | Override generation endpoint |
| `RAG_LLM_API_KEY` | — | For cloud LLM backends |
| `RAG_LLM_MAX_TOKENS` | `1024` | Max generation tokens |

Reindexing is not automatic; re-run `/api/v1/reindex` periodically (for example
from a cron job) so new audit records become searchable.
