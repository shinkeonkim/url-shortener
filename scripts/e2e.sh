#!/usr/bin/env sh
set -eu

work_dir="$(mktemp -d)"
port="${E2E_PORT:-18080}"
base="http://127.0.0.1:${port}"
cleanup() { kill "${server_pid:-0}" 2>/dev/null || true; rm -r "$work_dir"; }
trap cleanup EXIT INT TERM

go build -o "$work_dir/url-shortener" ./cmd/url-shortener
ADDRESS=":$port" DATABASE_PATH="$work_dir/test.db" BASE_DOMAIN=url.test \
  COOKIE_SECURE=false ADMIN_TOKEN=e2e-token "$work_dir/url-shortener" >"$work_dir/server.log" 2>&1 &
server_pid=$!

i=0
until curl -fsS "$base/health" >/dev/null 2>&1; do
  i=$((i + 1)); [ "$i" -lt 30 ] || { cat "$work_dir/server.log"; exit 1; }; sleep 0.2
done

curl -fsS -H 'Authorization: Bearer e2e-token' -H 'Content-Type: application/json' \
  -d '{"slug":"e2e","target_url":"https://example.com/result"}' "$base/api/v1/urls" | grep -q 'e2e.url.test'
[ "$(curl -sS -o /dev/null -w '%{http_code}' "$base/r/e2e")" = 302 ]
curl -fsS -H 'Authorization: Bearer e2e-token' "$base/api/v1/urls/e2e/stats" | grep -q '"clicks":1'
curl -fsS "$base/api/v1/urls/e2e/qr" -o "$work_dir/qr.png"
[ "$(wc -c < "$work_dir/qr.png")" -gt 100 ]
echo "E2E smoke test passed"
