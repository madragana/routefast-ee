#!/usr/bin/env bash
set -euo pipefail
COMPOSE=deployments/docker-compose/docker-compose.yml
DATABASE_URL="${DATABASE_URL:-postgres://yugabyte:yugabyte@localhost:5433/yugabyte}"

echo "==> Starting YugabyteDB"
docker compose -f "$COMPOSE" up -d yugabyte

echo "==> Waiting for database"
until psql "$DATABASE_URL" -c '\q' 2>/dev/null; do sleep 2; done

echo "==> Applying migrations"
for f in migrations/*.sql; do echo "  $f"; psql "$DATABASE_URL" -f "$f"; done

echo "==> Generating dev certificates"
go run ./cmd/gen-certs -out ./tls

echo "==> Done. Run 'make run' to start the server."
