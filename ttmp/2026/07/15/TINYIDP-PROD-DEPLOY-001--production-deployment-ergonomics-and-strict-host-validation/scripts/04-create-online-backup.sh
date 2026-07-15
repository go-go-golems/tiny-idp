#!/bin/sh
# Create and verify a consistent SQLite backup. It is safe to run while the
# strict host is serving because the backup command uses SQLite's online backup.
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
[ -f "$work_dir/idp.db" ] || { echo "missing $work_dir/idp.db" >&2; exit 1; }

backup_dir="$work_dir/backups"
backup_path="$backup_dir/idp-before-restore.db"
[ ! -e "$backup_path" ] || { echo "refusing to overwrite backup: $backup_path" >&2; exit 1; }
umask 077
mkdir -p "$backup_dir"
chmod 700 "$backup_dir"

go run ./cmd/tinyidp admin --db "$work_dir/idp.db" backup create --out "$backup_path"
go run ./cmd/tinyidp admin backup verify --path "$backup_path"

echo "verified backup: $backup_path"
echo "next: stop every tinyidp process using $work_dir/idp.db, then run scripts/05-offline-restore-drill.sh --work-dir $work_dir"
