# Architecture

## Role
`lipd-server` is the control plane for a fleet of autonomous RouteFast CE
nodes. CE nodes detect and mitigate attacks locally; the EE control plane gives
them identity, policy, and an auditable record — without sitting in the data
path.

## Components
- **API server** (`cmd/lipd-server`) — Go `net/http` server, TLS 1.3 + mTLS,
  7 REST/JSON endpoints.
- **Storage** (`internal/storage`) — `pgx/v5` pool against YugabyteDB.
- **Crypto** (`internal/crypto`) — ed25519 keypair generation, random bearer
  tokens, signature verification.
- **Cert tooling** (`cmd/gen-certs`) — dev-only self-signed CA + server/client
  certs.

## Why YugabyteDB
YugabyteDB speaks the PostgreSQL wire protocol, so the storage layer is plain
`pgx` and ordinary SQL — no proprietary client. We get distributed,
multi-region, horizontally scalable storage with synchronous replication, which
matches the "sovereign / mission-critical" posture of the Defence edition while
keeping a familiar Postgres surface. A node can fail without losing audit data.

## Data model
- `customers` — tenants and subscription tier.
- `nodes` — registered CE nodes with their active ed25519 public key.
- `tokens` — bearer tokens; refresh revokes (never deletes) the old one.
- `policies` — per-customer JSONB bundle + checksum.
- `audit_logs` — append-only decision/event records.
- `key_rotations` — append-only history of key rotations.

## Trust model
- Every connection is mutually authenticated (mTLS, TLS 1.3 only).
- Node-scoped calls also carry a short-lived bearer token bound to a node ID.
- Private keys are returned to the node exactly once and never persisted
  server-side beyond the public half.

## Request lifecycle (register)
1. Node presents a client cert → mTLS handshake.
2. `POST /register` → server generates ed25519 keypair, stores public key.
3. Server mints a 7-day token, fetches the customer's policy bundle.
4. Response returns node ID, keypair, token, policy checksum.
