#!/bin/bash
BASE=http://127.0.0.1:3080
KEY="X-ttpg-key: 12345"
BIN=${TTPG_BIN:-/tmp/opencode/ttpg_server}
if [ ! -x "$BIN" ]; then mkdir -p "$(dirname "$BIN")" && (cd "$(dirname "$0")/../.." && go build -o "$BIN" .); fi
PASS=0; FAIL=0

restart() { # env assignment string
  pkill -f 'ttpg_server' 2>/dev/null; sleep 1
  for i in $(seq 1 10); do pgrep -f 'ttpg_server' >/dev/null || break; sleep 0.3; done
  ( cd /tmp/opencode && setsid env PORT=3080 "$@" AUTH_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_auth?parseTime=true' MAIN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg?parseTime=true' PENGINAPAN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_penginapan?parseTime=true' USERS_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_users?parseTime=true' MONGO_URI='mongodb://127.0.0.1:27017' "$BIN" > /tmp/opencode/ttpg_server.log 2>&1 < /dev/null & )
  for i in $(seq 1 20); do
    curl -s -m 2 -o /dev/null "$BASE/auth/key" && return 0
    sleep 0.5
  done
  echo "server gagal start"; exit 1
}

t() { # name expect_reason body
  local name="$1" expect="$2" body="$3"
  local out code reason
  out=$(curl -s -m 10 -w "\n%{http_code}" -X POST "$BASE/gate" -H "$KEY" -H 'Content-Type: application/json' -d "$body")
  code=$(printf '%s' "$out" | tail -n1)
  reason=$(printf '%s' "$out" | sed '$d' | jq -r '.reason // empty')
  if [ "$code" = "200" ] && [ "$reason" = "$expect" ]; then
    PASS=$((PASS+1)); echo "PASS  $name -> $reason"
  elif [ "$code" = "200" ] && [ "$expect" = "FAILSIDE" ] && [ -n "$reason" ] && [ "${reason#VALID}" = "$reason" ]; then
    PASS=$((PASS+1)); echo "PASS  $name -> $reason (tidak lolos, sesuai harapan)"
  else
    FAIL=$((FAIL+1)); echo "FAIL  $name -> HTTP $code reason='$reason' (expect '$expect')"
  fi
}

echo "===== H. BYPASS=true ====="
restart BYPASS=true
t "noreg apa saja lolos" "BYPASS MODE" '{"noreg":"BELUMCEKSAMA","gate_id":9}'
t "noreg kedua juga lolos" "BYPASS MODE" '{"noreg":"BELUMCEK2","gate_id":9}'

echo "===== I. LIST_ID_CARD ====="
restart LIST_ID_CARD="PREMIUM01,PREMIUM02"
t "noreg di daftar -> LIST_ID_CARD" "VALID via LIST_ID_CARD" '{"noreg":"PREMIUM02","gate_id":3}'
t "noreg tidak di daftar -> fallback (bukan lolos)" "FAILSIDE" '{"noreg":"BUKANLIST","gate_id":3}'

echo "===== J. kembali normal ====="
restart
t "mode normal (SCAN001 tetap ada)" "VALID RESCAN" '{"noreg":"SCAN001","gate_id":1}'

echo ""
echo "================================"
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
echo "================================"
[ "$FAIL" = "0" ]
