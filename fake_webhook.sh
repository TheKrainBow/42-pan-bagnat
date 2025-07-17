#!/usr/bin/env bash
set -euo pipefail

# 1) Shared secret (must match what your backend is using)
SECRET="MokkoIsNotFat"

# 2) Determine payload: use first arg if given, otherwise default
if [ $# -eq 0 ]; then
  payload='{"eventType":"log","module_id":"auth","timestamp":"2025-07-17T16:00:00Z","payload":{"message":"Default test log"}}'
elif [ $# -eq 1 ]; then
  payload="$1"
else
  echo "Usage: $0 [ '<json-payload>' ]"
  exit 1
fi

# 3) Compute hex-encoded HMAC-SHA256 signature (no extra newline)
signature=$(printf '%s' "$payload" \
  | openssl dgst -sha256 -hmac "$SECRET" \
  | awk '{print $NF}')

# 4) Send the webhook
curl -v http://localhost:8080/webhooks/events \
  -H "Content-Type: application/json" \
  -H "X-Hook-Signature: $signature" \
  --data "$payload"
