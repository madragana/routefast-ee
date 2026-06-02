# RouteFast EE API Reference

Base URL: `https://<host>:8443/api/v1`
All endpoints require **mTLS client certificates** at the transport layer and
return **JSON**. Node-scoped endpoints additionally require a bearer token in
the `Authorization: Bearer <token>` header.

---

## 1. Health — `GET /health`

Liveness probe and database connectivity check. mTLS only.

**200 OK**
```json
{ "status": "healthy", "uptime": 1716835200, "database": "ok" }
```

---

## 2. Register node — `POST /register`

Provisions a new CE node. The server generates an ed25519 keypair, stores the
public half, issues a first 7-day token, and returns the private key **once**.

**Request**
```json
{ "customer_id": "11111111-1111-1111-1111-111111111111", "initial_secret": "..." }
```

**201 Created**
```json
{
  "node_id": "…",
  "public_key": "base64…",
  "private_key": "base64…",
  "token": "…",
  "expires_at": "2026-06-09T00:00:00Z",
  "policies_checksum": "…"
}
```

---

## 3. Refresh token — `POST /token/refresh`

Revokes the presented (valid, unexpired) token and returns a fresh one for the
same node. Nodes refresh automatically around day 6.

**Headers:** `Authorization: Bearer <current-token>`

**200 OK**
```json
{ "token": "…", "expires_at": "2026-06-16T00:00:00Z" }
```

---

## 4. Rotate key — `POST /key/rotate`

Generates a new ed25519 keypair for the authenticated node, records the
rotation in `key_rotations` (append-only), and updates the node's active key.

**Headers:** `Authorization: Bearer <token>`

**200 OK**
```json
{ "public_key": "base64…", "private_key": "base64…", "rotated_at": "…" }
```

---

## 5. Get policies — `GET /policies?customer_id=<id>`

Returns the active policy bundle (quorum rules, mitigation templates, evidence
schemas). The `X-Policy-Checksum` response header lets nodes validate freshness.

**200 OK** — the raw policy JSON document.

---

## 6. Log decision — `POST /audit/log`

Appends a decision/event to the append-only audit log.

**Headers:** `Authorization: Bearer <token>`

**Request**
```json
{
  "customer_id": "…",
  "event_type": "flowspec_drop",
  "detail": { "rule": "rf-2019", "duration_s": 300 }
}
```

**201 Created**
```json
{ "id": "…" }
```

---

## 7. Audit trail — `GET /audit/trail?customer_id=<id>&start_date=YYYY-MM-DD&end_date=YYYY-MM-DD`

Returns audit records for a customer within a date range, ordered by time
ascending. Used for SOC 2 / HIPAA compliance reporting.

**200 OK**
```json
{ "customer_id": "…", "count": 42, "entries": [ { "id": "…", "event_type": "…", "logged_at": "…" } ] }
```

---

## Error format

All errors return JSON: `{ "error": "human-readable message" }` with the
appropriate HTTP status (400, 401, 404, 500).
