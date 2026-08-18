#!/bin/sh
# LaundryPro restore script.
#
# Restores PostgreSQL and Redis from a backup directory created by
# scripts/backup.sh. RESTORE ORDER MATTERS: Postgres first, then Redis.
#
# Usage: ./scripts/restore.sh <backup-dir>
#   e.g. ./scripts/restore.sh backups/20260810T030000Z

set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <backup-dir>" >&2
  exit 1
fi
SRC="$1"
DB_SQL="$SRC/db.sql"
RDB="$SRC/redis.rdb"

[ -f "$DB_SQL" ] || { echo "missing $DB_SQL" >&2; exit 1; }
[ -f "$RDB" ] || { echo "missing $RDB" >&2; exit 1; }

PG_CT="${POSTGRES_CONTAINER:-laundry-pro-postgres}"
REDIS_CT="${REDIS_CONTAINER:-laundry-pro-redis}"
PG_USER="${POSTGRES_USER:-aeroxe}"
PG_DB="${POSTGRES_DB:-laundry_pro}"

echo "[restore] postgres (drop/recreate + load $DB_SQL)"
# DROP fails if any connection to $PG_DB is active; stop the app first or
# terminate connections before restoring.
docker exec "$PG_CT" psql -U "$PG_USER" -c "DROP DATABASE IF EXISTS $PG_DB" >/dev/null
docker exec "$PG_CT" psql -U "$PG_USER" -c "CREATE DATABASE $PG_DB" >/dev/null
docker exec -i "$PG_CT" psql -U "$PG_USER" -d "$PG_DB" < "$DB_SQL"

echo "[restore] redis ($RDB)"
docker exec "$REDIS_CT" redis-cli CONFIG SET dir /data >/dev/null
docker cp "$RDB" "$REDIS_CT:/data/dump.rdb"
docker exec "$REDIS_CT" redis-cli CONFIG SET dbfilename dump.rdb >/dev/null
docker exec "$REDIS_CT" redis-cli SHUTDOWN NOSAVE >/dev/null 2>&1 || true
docker start "$REDIS_CT" >/dev/null

echo "[restore] complete; verify with /api/v1/health/ready"
