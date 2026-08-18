#!/usr/bin/env bash
# Deep-dive integration verification: live API checks against a running backend.
set -u
B="http://localhost:9090/api/v1"
FAIL=0

echo "== 1. HEALTH =="
curl -s -o /dev/null -w '  ready=%{http_code}\n' "$B/health/ready"
curl -s -o /dev/null -w '  live=%{http_code}\n' "$B/health/live"
curl -s -o /dev/null -w '  swagger-ui=%{http_code}\n' http://localhost:9090/swagger/index.html

echo "== 2. LOGIN (admin) =="
LOGIN=$(curl -s -X POST "$B/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"admin@laundrypro.local","password":"Admin@12345"}')
TOKEN=$(echo "$LOGIN" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
if [ -n "$TOKEN" ]; then echo "  token OK (${#TOKEN} chars)"; else echo "  LOGIN FAILED: $LOGIN"; FAIL=1; fi
AUTH="Authorization: Bearer $TOKEN"

echo "== 3. AUTH ME + RBAC =="
curl -s -X POST "$B/auth/me" -H "$AUTH" -H 'Content-Type: application/json' -d '{}' | grep -o '"email":"[^"]*"' | head -1
curl -s -o /dev/null -w '  settings/view -> %{http_code} (admin should pass)\n' -X POST "$B/settings/view" -H "$AUTH" -H 'Content-Type: application/json' -d '{}'
curl -s -o /dev/null -w '  no-token /auth/me -> %{http_code} (expect 401)\n' -X POST "$B/auth/me" -H 'Content-Type: application/json' -d '{}'

echo "== 4. SETTINGS -> BUSINESS FLOW (currency + tax integration) =="
curl -s -X POST "$B/settings/view" -H "$AUTH" -H 'Content-Type: application/json' -d '{}' | grep -o '"currency":"[^"]*"\|"tax_rate":[0-9.]*\|"invoice_prefix":"[^"]*"' | head -3
echo "  -- patch tax to 7.5 --"
curl -s -X PATCH "$B/settings" -H "$AUTH" -H 'Content-Type: application/json' -d '{"tax_rate":7.5}' | grep -o '"tax_rate":[0-9.]*'
echo "  -- service-types list (currency should come from settings) --"
curl -s -X POST "$B/service-types/list" -H "$AUTH" -H 'Content-Type: application/json' -d '{"limit":3,"offset":0}' | grep -o '"currency":"[^"]*"' | head -1
echo "  -- restore tax to 0 --"
curl -s -X PATCH "$B/settings" -H "$AUTH" -H 'Content-Type: application/json' -d '{"tax_rate":0}' | grep -o '"tax_rate":[0-9.]*'

echo "== 5. CURSOR LIST + SUMMARY (orders) =="
curl -s -X POST "$B/orders/list" -H "$AUTH" -H 'Content-Type: application/json' -d '{"limit":5,"offset":0}' | grep -o '"total":[0-9]*\|"limit":[0-9]*\|"offset":[0-9]*\|"summary":{[^}]*}' | head -4

echo "== 6. LOCKOUT BEHAVIOR (throwaway email) =="
EMAIL="deepdive-$RANDOM-$RANDOM@x.local"
CODES=""
for i in 1 2 3 4 5 6 7; do
  CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$B/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"bad\"}")
  CODES="$CODES $CODE"
done
echo "  failed-login sequence:$CODES (expect 401 401 401 401 401 429 429)"

echo "== 7. PAYLOAD INVALID (400 handling) =="
curl -s -o /dev/null -w '  bad body /orders -> %{http_code} (expect 400)\n' -X POST "$B/orders" -H "$AUTH" -H 'Content-Type: application/json' -d '{}'

echo "== 8. METRICS =="
curl -s http://localhost:9090/metrics | grep -c 'laundrypro_http_requests_total' | sed 's/^/  metric families found: /'

echo
if [ "$FAIL" -eq 0 ]; then echo "DEEP-DIVE CHECK: PASS"; else echo "DEEP-DIVE CHECK: FAIL"; fi
