#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:9999/api}"
DURATION="${DURATION:-30}"
SLEEP="${SLEEP:-0.1}"

KEYS=(Tom Jack Sam Alice Bob Eve Mike Lily Rose David Jenny Leo)

end=$((SECONDS + DURATION))
count=0

while [ "$SECONDS" -lt "$end" ]; do
  key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
  curl -s "${BASE_URL}?key=${key}" >/dev/null || true
  count=$((count + 1))
  if (( count % 50 == 0 )); then
    echo "sent ${count} requests..."
  fi
  sleep "$SLEEP"
done

echo "done: ${count} requests in ${DURATION}s"
