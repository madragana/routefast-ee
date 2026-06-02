# Quick Start

## Prerequisites
- Docker + Docker Compose
- Go 1.22+
- PostgreSQL client (`psql`)

## One-command setup
```bash
make setup     # starts YugabyteDB, applies migrations, generates dev certs
make run       # builds and runs the server (mTLS on :8443)
```

## Manual setup
```bash
# 1. Start YugabyteDB
docker compose -f deployments/docker-compose/docker-compose.yml up -d yugabyte

# 2. Apply schema
for f in migrations/*.sql; do
  psql postgres://yugabyte:yugabyte@localhost:5433/yugabyte -f "$f"
done

# 3. Generate dev mTLS certs
go run ./cmd/gen-certs -out ./tls

# 4. Environment
export DATABASE_URL="postgres://yugabyte:yugabyte@localhost:5433/yugabyte"
export TLS_CERT_FILE="./tls/server.crt"
export TLS_KEY_FILE="./tls/server.key"
export TLS_CLIENT_CA_FILE="./tls/ca.crt"

# 5. Build & run
go build -o bin/lipd-server ./cmd/lipd-server
./bin/lipd-server
```

## Smoke test
```bash
curl --cacert tls/ca.crt --cert tls/client.crt --key tls/client.key \
  https://localhost:8443/api/v1/health
# {"status":"healthy","uptime":...,"database":"ok"}
```
See `examples/curl-examples.sh` for register → policies → audit calls.
