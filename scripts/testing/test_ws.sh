#!/bin/bash
BASE=${BASE:-http://127.0.0.1:8090}
WSHOST=$(printf '%s' "$BASE" | sed 's|^http|ws|')
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
CLIENT=/tmp/opencode/ws_client
PASS=0; FAIL=0

if [ ! -x "$CLIENT" ]; then
  (cd "$ROOT" && go build -o "$CLIENT" ./scripts/testing/wsclient) || { echo "gagal build wsclient"; exit 1; }
fi

check() {
  if [ "$3" = "$2" ]; then PASS=$((PASS+1)); echo "PASS  $1"
  else FAIL=$((FAIL+1)); echo "FAIL  $1 (got '$3', expect '$2')"; fi
}

TOKEN=$(curl -s -m 5 -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token // empty')
if [ -n "$TOKEN" ]; then check "login -> token" "OK" "OK"; else check "login -> token" "OK" "FAIL"; fi

echo "===== WS. AUTENTIKASI ====="
NOAUTH=$(curl -s -m 5 -o /dev/null -w "%{http_code}" "$BASE/ws")
check "tanpa token -> 401" "401" "$NOAUTH"

BADAUTH=$(curl -s -m 5 -o /dev/null -w "%{http_code}" "$BASE/ws?token=salah")
check "token salah -> 401" "401" "$BADAUTH"

echo "===== WS. SETUP DATA ====="
SUF=$(date +%s)
LOC_ID=$(curl -s -X POST "$BASE/penginapan/locations" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"name\":\"WS $SUF\",\"keterangan\":\"ws test\"}" | jq -r '.data.id // empty')
check "POST location" "OK" "$([ -n "$LOC_ID" ] && echo OK || echo FAIL)"

R1=$(curl -s -X POST "$BASE/penginapan/rooms" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"location_id\":$LOC_ID,\"room_number\":\"WS1-$SUF\",\"room_type\":\"pria\",\"capacity\":1}" | jq -r '.data.id // empty')
R2=$(curl -s -X POST "$BASE/penginapan/rooms" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"location_id\":$LOC_ID,\"room_number\":\"WS2-$SUF\",\"room_type\":\"pria\",\"capacity\":1}" | jq -r '.data.id // empty')
check "POST room R1" "OK" "$([ -n "$R1" ] && echo OK || echo FAIL)"
check "POST room R2" "OK" "$([ -n "$R2" ] && echo OK || echo FAIL)"

echo "===== WS. EVENT ROOM (status change) ====="
OUT=/tmp/opencode/ws_test_out.txt
"$CLIENT" "$WSHOST/ws?token=$TOKEN" 8s > "$OUT" 2>&1 &
CPID=$!
sleep 1
grep -q "CONNECTED" "$OUT"; check "client terhubung" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"

curl -s -X PUT "$BASE/penginapan/rooms/$R1/status" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"maintenance"}' > /dev/null
sleep 2
kill $CPID 2>/dev/null; wait $CPID 2>/dev/null

grep -q 'EVENT type=room' "$OUT"; check "event room terkirim" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"
grep -q '"action":"status"' "$OUT"; check "action=status" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"
grep -q '"status":"maintenance"' "$OUT"; check "payload maintenance" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"

echo "===== WS. EVENT CHECK-IN (occupancy + guest) ====="
"$CLIENT" "$WSHOST/ws?token=$TOKEN" 8s > "$OUT" 2>&1 &
CPID=$!
sleep 1
CID=$(curl -s -X POST "$BASE/penginapan/guests" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R2,\"name\":\"WS Test\",\"gender\":\"pria\",\"harga\":10000}],\"payment\":{\"amount\":10000,\"payment_method_id\":1,\"source_id\":1}}" | jq -r '.data.guests[0].id // empty')
sleep 2
kill $CPID 2>/dev/null; wait $CPID 2>/dev/null

check "check-in tamu" "OK" "$([ -n "$CID" ] && echo OK || echo FAIL)"
grep -q '"action":"occupancy"' "$OUT"; check "event occupancy" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"
grep -q 'EVENT type=guest' "$OUT"; check "event guest" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"
grep -q '"action":"checkin"' "$OUT"; check "action=checkin" "OK" "$([ $? = 0 ] && echo OK || echo FAIL)"

if [ -n "$CID" ]; then
  curl -s -X POST "$BASE/penginapan/guests/$CID/checkout" -H "Authorization: Bearer $TOKEN" > /dev/null
fi

echo ""
echo "================================"
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
echo "================================"
[ "$FAIL" = "0" ]
