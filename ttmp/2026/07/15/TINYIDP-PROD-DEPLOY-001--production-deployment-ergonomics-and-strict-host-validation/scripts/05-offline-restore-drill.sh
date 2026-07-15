#!/bin/sh
# Prove a verified backup can replace a stopped database and preserve a
# rollback artifact. This script intentionally adds a disposable user after
# the backup, then proves restoration removes it while the rollback keeps it.
set -eu

usage() {
  echo "usage: $0 --work-dir ABSOLUTE_PATH" >&2
  exit 64
}

work_dir=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --work-dir) [ "$#" -ge 2 ] || usage; work_dir=$2; shift 2 ;;
    *) usage ;;
  esac
done
[ -n "$work_dir" ] || usage
case "$work_dir" in /*) ;; *) echo "--work-dir must be absolute" >&2; exit 64 ;; esac

db_path="$work_dir/idp.db"
backup_path="$work_dir/backups/idp-before-restore.db"
for path in "$db_path" "$backup_path"; do
  [ -f "$path" ] || { echo "missing required file: $path" >&2; exit 1; }
done
for sidecar in "$db_path-wal" "$db_path-shm"; do
  [ ! -e "$sidecar" ] || { echo "refusing restore while SQLite sidecar exists: $sidecar; stop tinyidp and wait for it to close the database" >&2; exit 1; }
done

probe_login=restore-probe
if go run ./cmd/tinyidp admin --db "$db_path" user get --login "$probe_login" >/dev/null 2>&1; then
  echo "refusing to reuse existing restore probe user: $probe_login" >&2
  exit 1
fi
printf '%s\n' 'Restore Drill Password 2026!' | go run ./cmd/tinyidp admin --db "$db_path" user create --login "$probe_login" --name 'Restore Drill Probe' --password-from-stdin
go run ./cmd/tinyidp admin --db "$db_path" user get --login "$probe_login" >/dev/null

go run ./cmd/tinyidp admin --db "$db_path" backup restore --path "$backup_path"
go run ./cmd/tinyidp admin --db "$db_path" doctor
if go run ./cmd/tinyidp admin --db "$db_path" user get --login "$probe_login" >/dev/null 2>&1; then
  echo "restore drill failed: post-backup probe user still exists" >&2
  exit 1
fi

rollback_path=
for candidate in "$db_path".pre-restore-*; do
  [ -f "$candidate" ] || continue
  rollback_path=$candidate
  break
done
[ -n "$rollback_path" ] || { echo "restore drill failed: rollback database was not preserved" >&2; exit 1; }
go run ./cmd/tinyidp admin --db "$rollback_path" user get --login "$probe_login" >/dev/null

echo "restore drill passed"
echo "restored database: $db_path"
echo "preserved rollback: $rollback_path"
