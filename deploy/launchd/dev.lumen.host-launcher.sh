#!/bin/sh
set -eu
data_dir="/Users/your-user/Library/Application Support/Lumen"
attempt_file="$data_dir/.launchd-failures"
mkdir -p "$data_dir"
chmod 700 "$data_dir"
set +e
/usr/local/bin/lumen-host serve
status=$?
set -e
if [ "$status" -eq 0 ]; then
  rm -f "$attempt_file"
  exit 0
fi
attempt=0
if [ -r "$attempt_file" ]; then attempt=$(cat "$attempt_file"); fi
attempt=$((attempt + 1))
printf '%s\n' "$attempt" > "$attempt_file"
chmod 600 "$attempt_file"
if [ "$attempt" -ge 3 ]; then exit 0; fi
exit "$status"
