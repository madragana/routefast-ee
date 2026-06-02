#!/usr/bin/env bash
set -euo pipefail
DATABASE_URL="${DATABASE_URL:-postgres://yugabyte:yugabyte@localhost:5433/yugabyte}"
for f in migrations/*.sql; do echo "applying $f"; psql "$DATABASE_URL" -f "$f"; done
