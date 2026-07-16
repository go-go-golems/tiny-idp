#!/usr/bin/env bash
set -euo pipefail

# Starts the complete development application in tmux. In a second terminal,
# run the printed device-login command, open the verification URL in a real
# browser, approve with the seeded account, then run bbs-post/bbs-get.
#
# Usage from repository root:
#   ttmp/.../scripts/run-xapp-device-smoke.sh
#
# The script intentionally does not automate approval or print bearer tokens.

session="tinyidp-xapp-device-smoke"
listen="127.0.0.1:18878"
base_url="http://${listen}"
state_root="/tmp/tinyidp-xapp-device-smoke-18878"
cache_file="${state_root}/device-token.json"

if tmux has-session -t "${session}" 2>/dev/null; then
  printf 'tmux session %s already exists; inspect or stop it before rerunning.\n' "${session}" >&2
  exit 1
fi

tmux new-session -d -s "${session}" \
  "go run ./cmd/tinyidp-xapp serve --listen ${listen} --public-base-url ${base_url} --state-root ${state_root} --login alice --password 'correct horse battery staple' --second-login bob --second-password 'correct horse battery staple'; exec zsh"

printf 'Server session: %s\n' "${session}"
printf 'Watch logs: tmux capture-pane -pt %s\n' "${session}"
printf 'Readiness probe: curl -fsS %s/idp/.well-known/openid-configuration\n' "${base_url}"
printf 'Device login: go run ./cmd/tinyidp-xapp device-login --issuer %s/idp --audience %s/api --token-cache %s\n' "${base_url}" "${base_url}" "${cache_file}"
printf 'Post: go run ./cmd/tinyidp-xapp bbs-post --api-base-url %s --token-cache %s --title "Device dispatch" --body "Posted after real device approval" --category notes\n' "${base_url}" "${cache_file}"
printf 'Read: go run ./cmd/tinyidp-xapp bbs-get --api-base-url %s --token-cache %s\n' "${base_url}" "${cache_file}"
