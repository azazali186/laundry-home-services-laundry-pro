#!/bin/sh
set -e

# Apply pending schema migrations before serving traffic. The runner records
# applied versions in schema_migrations, so this is safe on every boot and
# idempotent across scaled-out replicas. Fail fast if a migration fails.
echo "[entrypoint] applying schema migrations"
/app/server -migrate-only

echo "[entrypoint] starting server"
exec /app/server
