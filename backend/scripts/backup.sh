#!/bin/sh
# LaundryPro backup script.
#
# Backs up PostgreSQL (pg_dump, plain SQL) and Redis (SAVE + copy RDB) into a
# timestamped directory, pruning backups older than BACKUP_KEEP_DAYS (default 14).
#
# Usage:   ./scripts/backup.sh
# Crontab: 0 3 * * * /path/to/laundry-pro/backend/scripts/backup.sh >> /var/log/laundry-backup.log 2>&1
#
# Env (defaults match docker-compose):
#   BACKUP_DIR  - destination directory (default ./backups)
#   BACKUP_KEEP_DAYS - retention in days (default 14)
#   POSTGRES_CONTAINER - container name (default laundry-pro-postgres)
#   POSTGRES_USER / POSTGRES_DB
#   REDIS_CONTAINER - container name (default laundry-pro-redis)

set -eu

BACKUP_DIR="${BACKUP_DIR:-$(dirname "$0")/../backups}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
PG_CT="${POSTGRES_CONTAINER:-laundry-pro-postgres}"
REDIS_CT="${REDIS_CONTAINER:-laundry-pro-redis}"
PG_USER="${POSTGRES_USER:-aeroxe}"
PG_DB="${POSTGRES_DB:-laundry_pro}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DEST="$BACKUP_DIR/$STAMP"
mkdir -p "$DEST"

echo "[backup] postgres dump -> $DEST/db.sql"
docker exec "$PG_CT" pg_dump -U "$PG_USER" -d "$PG_DB" --no-owner > "$DEST/db.sql"

echo "[backup] redis snapshot"
docker exec "$REDIS_CT" redis-cli SAVE >/dev/null
docker cp "$REDIS_CT:/data/dump.rdb" "$DEST/redis.rdb"

echo "[backup] done: $DEST"
echo "$STAMP" > "$BACKUP_DIR/latest"

# Prune old backups.
OLD="$(find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -mtime +"$KEEP_DAYS" 2>/dev/null)"
if [ -n "$OLD" ]; then
  echo "[backup] pruning:"
  echo "$OLD"
  echo "$OLD" | xargs rm -rf
fi
