#!/usr/bin/env bash
# Backups of the hosted database (docs/ONLINE_AND_RULES_PLAN.md, Phase 6, step 7).
#
#   scripts/hosted-db.sh backup         Download a dump to db/backups/hosted-<time>.dump,
#                                       with its tables' row counts beside it (.counts).
#   scripts/hosted-db.sh check FILE     Restore FILE into a throwaway Postgres and compare
#                                       its row counts with the ones saved at backup time.
#   scripts/hosted-db.sh restore FILE   Replace the hosted database with FILE (asks first).
#
# HOSTED_DATABASE_URL (the Render database's External URL, in .env) is passed to the
# Postgres 18 tools through the environment and never printed. Render also keeps 3 days
# of point-in-time restore; these dumps are the copy on the GM's own machine.
set -euo pipefail

PG_IMAGE=postgres:18
BACKUP_DIR=db/backups
# Each table's row count, one "table|count" line per table.
COUNTS_SQL="select table_name, (xpath('/row/c/text()', query_to_xml(format('select count(*) as c from public.%I', table_name), false, true, '')))[1]::text
  from information_schema.tables where table_schema = 'public' and table_type = 'BASE TABLE' order by 1"

die() { echo "$*" >&2; exit 1; }

need_url() {
  [ -n "${HOSTED_DATABASE_URL:-}" ] || die "Set HOSTED_DATABASE_URL in .env (the database's External URL)."
}

# pg runs a Postgres 18 tool against the hosted database; stdin and stdout pass through.
pg() {
  docker run --rm -i -e PGURL="$HOSTED_DATABASE_URL" "$PG_IMAGE" sh -c "$1"
}

backup() {
  need_url
  mkdir -p "$BACKUP_DIR"
  local file
  file="$BACKUP_DIR/hosted-$(date +%Y%m%d-%H%M%S).dump"
  umask 077
  pg 'pg_dump -Fc --no-owner --no-privileges "$PGURL"' > "$file.part"
  pg "psql \"\$PGURL\" -Atc \"$COUNTS_SQL\"" > "$file.counts"
  mv "$file.part" "$file"
  echo "Saved $file ($(du -h "$file" | cut -f1)), $(wc -l < "$file.counts" | tr -d ' ') tables."
}

check() {
  local file=${1:-}
  [ -f "$file" ] || die "Usage: $0 check FILE (a .dump from backup)"
  [ -f "$file.counts" ] || die "No $file.counts beside the dump."
  local name="dce-restore-check-$$"
  docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=check "$PG_IMAGE" > /dev/null
  # shellcheck disable=SC2064 # stop this container, named now, whenever the script ends
  trap "docker stop $name > /dev/null 2>&1 || true" EXIT
  for _ in $(seq 1 30); do
    docker exec "$name" pg_isready -U postgres -q 2> /dev/null && break
    sleep 1
  done
  docker exec "$name" createdb -U postgres check
  docker exec -i "$name" pg_restore -U postgres -d check --no-owner --no-privileges --exit-on-error < "$file"
  local restored
  restored=$(docker exec "$name" psql -U postgres -d check -Atc "$COUNTS_SQL")
  if [ "$restored" = "$(cat "$file.counts")" ]; then
    echo "Restored $file into a throwaway database: every table's row count matches the backup."
    echo "$restored" | sed 's/|/: /; s/^/  /'
  else
    echo "Row counts differ from the backup:" >&2
    diff <(cat "$file.counts") <(echo "$restored") >&2 || true
    exit 1
  fi
}

restore() {
  need_url
  local file=${1:-}
  [ -f "$file" ] || die "Usage: $0 restore FILE (a .dump from backup)"
  echo "This REPLACES everything in the hosted database with $file."
  echo "Prefer Render's point-in-time restore for anything in the last 3 days."
  read -r -p "Type replace to go on: " answer
  [ "$answer" = replace ] || die "Nothing changed."
  pg 'psql "$PGURL" -v ON_ERROR_STOP=1 -qc "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"'
  pg 'pg_restore --no-owner --no-privileges --exit-on-error -d "$PGURL"' < "$file"
  echo "Restored. Redeploy the web service (make deploy, or the deploy hook) so it reconnects and migrates."
}

case "${1:-}" in
  backup) backup ;;
  check) check "${2:-}" ;;
  restore) restore "${2:-}" ;;
  *) die "Usage: $0 backup | check FILE | restore FILE" ;;
esac
