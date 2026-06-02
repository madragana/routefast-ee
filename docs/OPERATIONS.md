# Operations

## Configuration (environment)
| Var | Default | Meaning |
|-----|---------|---------|
| `LISTEN_ADDR` | `:8443` | mTLS bind address |
| `DATABASE_URL` | `postgres://yugabyte:yugabyte@localhost:5433/yugabyte` | YugabyteDB DSN |
| `TLS_CERT_FILE` | `./tls/server.crt` | Server certificate |
| `TLS_KEY_FILE` | `./tls/server.key` | Server private key |
| `TLS_CLIENT_CA_FILE` | `./tls/ca.crt` | CA used to verify client certs |

## Deploy
- **Container**: `make docker-build` then push (the release workflow does this
  on a `v*` tag, publishing to GHCR).
- **Kubernetes**: Helm chart under `deployments/kubernetes/helm` (to be filled
  in for your cluster).
- **Cloud infra**: Terraform stubs under `deployments/terraform/{aws,gcp,azure}`.

## Migrations
Run `scripts/migrate.sh` (or `make migrate`). Migrations are ordered and
idempotent (`IF NOT EXISTS`).

## Health & monitoring
- Liveness: `GET /api/v1/health` returns `database: ok|error`.
- Logs: one line per request (method, path, latency) on stdout.

## Backups
YugabyteDB handles replication; schedule `ysql_dump` or distributed backups per
your DR policy. The audit log is the system of record for compliance — protect
it accordingly.
