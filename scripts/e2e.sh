#!/usr/bin/env bash
# Builds the binary, seeds demo data into a temp dir, and walks an order through the API.
set -euo pipefail
export PATH=/opt/homebrew/bin:$PATH
cd "$(dirname "$0")/.."

T=$(mktemp -d)
PORT=${PORT:-8098}
BIN=bin/deepcuts
DB=$T/e2e.sqlite
JAR=$T/office.jar
DJAR=$T/driver.jar
API=http://localhost:$PORT/api
# Match the server's business timezone (DEEPCUTS_TIMEZONE, default America/New_York)
# rather than the system zone, so TODAY agrees with what the server considers "today".
export DEEPCUTS_TIMEZONE=UTC
TODAY=$(date -u +%F)

make -s build >/dev/null
DEEPCUTS_DB_PATH=$DB DEEPCUTS_DATA_DIR=$T $BIN seed demo >/dev/null
DEEPCUTS_DB_PATH=$DB DEEPCUTS_DATA_DIR=$T DEEPCUTS_ADDR=:$PORT $BIN serve >$T/server.log 2>&1 &
SERVER=$!
trap 'kill $SERVER 2>/dev/null; rm -rf "$T"' EXIT
for i in $(seq 1 50); do curl -s -o /dev/null "$API/driver/drivers" && break; sleep 0.1; done

# call METHOD PATH [JSON] [JAR] -> prints the body; the HTTP status goes to $T/status so it
# survives the $(...) subshell that callers use.
call() {
  local m=$1 p=$2 body=${3:-} jar=${4:-$JAR}
  if [ -n "$body" ]; then
    curl -s -o "$T/body" -w '%{http_code}' -b "$jar" -c "$jar" -X "$m" "$API$p" -H 'Content-Type: application/json' -d "$body" > "$T/status"
  else
    curl -s -o "$T/body" -w '%{http_code}' -b "$jar" -c "$jar" -X "$m" "$API$p" > "$T/status"
  fi
  cat "$T/body"
}
expect() { local st; st=$(cat "$T/status"); if [ "$st" != "$1" ]; then echo "FAIL: $2 → HTTP $st: $3" >&2; exit 1; fi; echo "ok: $2"; }
jq_() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)"; }

B=$(call POST /office/login '{"email":"office@demo.local","password":"demo1234"}'); expect 200 "office login" "$B"
CUST=$(call GET /office/customers | jq_ 'd[0]["id"]')
PROD=$(call GET /office/products | jq_ '[p for p in d if p["catchWeight"]][0]["id"]')
B=$(call POST /office/orders "{\"customerId\":$CUST,\"requestedDeliveryDate\":\"$TODAY\"}"); expect 201 "create order" "$B"
ORDER=$(echo "$B" | jq_ 'd["id"]')
B=$(call POST /office/orders/$ORDER/lines "{\"productId\":$PROD,\"orderedQty\":200}"); expect 200 "add line" "$B"
LINE=$(echo "$B" | jq_ 'd["lines"][0]["id"]')
[ "$(echo "$B" | jq_ 'd["lines"][0]["amountSource"]')" = "estimated" ] || { echo "FAIL: amount source"; exit 1; }
B=$(call POST /office/orders/$ORDER/confirm); expect 200 "confirm" "$B"
B=$(call PATCH /office/orders/$ORDER/lines/$LINE '{"shippedWeight":11850}'); expect 200 "shipped weight" "$B"
DRIVER=$(call GET /office/drivers | jq_ '[x for x in d if x["displayName"]=="Riley"][0]["id"]')
B=$(call POST /office/routes "{\"routeDate\":\"$TODAY\",\"driverUserId\":$DRIVER,\"truckLabel\":\"e2e\"}"); expect 201 "create route" "$B"
ROUTE=$(echo "$B" | jq_ 'd["id"]')
B=$(call POST /office/routes/$ROUTE/stops "{\"orderId\":$ORDER}"); expect 200 "add stop" "$B"
STOP=$(echo "$B" | jq_ 'd["stops"][0]["id"]')
B=$(call POST /office/routes/$ROUTE/out); expect 200 "route out" "$B"

B=$(call POST /driver/login "{\"userId\":$DRIVER,\"pin\":\"222222\"}" "$DJAR"); expect 200 "driver login" "$B"
B=$(call GET /driver/route "" "$DJAR"); expect 200 "driver route" "$B"
[ "$(echo "$B" | jq_ 'd["stops"][0]["stop"]["id"]')" = "$STOP" ] || { echo "FAIL: driver sees wrong stop"; exit 1; }
CID=$(python3 -c 'import uuid; print(uuid.uuid4())')
B=$(call POST /driver/stops/$STOP/actions "{\"clientId\":\"$CID\",\"type\":\"deliver\",\"proof\":{\"type\":\"name\",\"name\":\"Pat\"},\"lines\":[{\"lineId\":$LINE,\"deliveredWeight\":6000,\"shortageNote\":\"one case short\"}]}" "$DJAR"); expect 200 "deliver" "$B"
[ "$(echo "$B" | jq_ 'd["applied"]')" = "True" ] || { echo "FAIL: not applied"; exit 1; }
B=$(call POST /driver/stops/$STOP/actions "{\"clientId\":\"$CID\",\"type\":\"deliver\",\"proof\":{\"type\":\"name\",\"name\":\"Pat\"}}" "$DJAR"); expect 200 "replay" "$B"
[ "$(echo "$B" | jq_ 'd["applied"]')" = "False" ] || { echo "FAIL: replay applied twice"; exit 1; }
B=$(call POST /driver/routes/$ROUTE/complete "" "$DJAR"); expect 200 "complete route" "$B"

B=$(call GET /office/orders/$ORDER); expect 200 "order after delivery" "$B"
[ "$(echo "$B" | jq_ 'd["status"]')" = "delivered" ] || { echo "FAIL: status $(echo "$B" | jq_ 'd["status"]')"; exit 1; }
[ "$(echo "$B" | jq_ 'd["needsReview"]')" = "True" ] || { echo "FAIL: needsReview"; exit 1; }
[ "$(echo "$B" | jq_ 'd["lines"][0]["amountSource"]')" = "delivered" ] || { echo "FAIL: amount source after delivery"; exit 1; }
B=$(call POST /office/orders/$ORDER/finalize); expect 200 "finalize" "$B"
[ "$(echo "$B" | jq_ 'd["status"]')" = "finalized" ] || { echo "FAIL: not finalized"; exit 1; }
echo "E2E PASSED"
