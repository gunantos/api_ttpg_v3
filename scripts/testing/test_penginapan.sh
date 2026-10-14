#!/bin/bash
BASE=${BASE:-http://127.0.0.1:3080}
PASS=0; FAIL=0
BODY=""; CODE=""

req() {
  local name="$1" method="$2" path="$3" expect="$4"; shift 4
  BODY=$(curl -s -m 15 -w "\n%{http_code}" -X "$method" "$BASE$path" "$@" 2>/dev/null)
  CODE=$(printf '%s' "$BODY" | tail -n1)
  BODY=$(printf '%s' "$BODY" | sed '$d')
  if [ "$CODE" != "$expect" ]; then
    echo "FAIL  $name -> HTTP $CODE (expect $expect) body=${BODY:0:220}"
    FAIL=$((FAIL+1)); return 1
  fi
  PASS=$((PASS+1)); echo "PASS  $name (HTTP $CODE)"
}

ja() {
  local name="$1" expr="$2" got
  got=$(printf '%s' "$BODY" | jq -r "$expr" 2>/dev/null)
  if [ "$got" != "true" ]; then
    echo "FAIL  $name -> jq '$expr' = '${got}' body=${BODY:0:300}"
    FAIL=$((FAIL+1)); return 1
  fi
  PASS=$((PASS+1)); echo "PASS  $name"
}

jget() { printf '%s' "$BODY" | jq -r "$1"; }

echo "===== A. AUTH ====="
req "login -> 200" POST /auth/login 200 -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'
TOKEN=$(jget .data.token)
req "configs tanpa token -> 401" GET /penginapan/configs 401

echo "===== B. CONFIGS ====="
req "POST config -> 201" POST /penginapan/configs 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"key_configs":"tarif_default","value_configs":"100000"}'
CONFIG_ID=$(jget .data.id_configs)
req "GET configs list" GET /penginapan/configs 200 -H "Authorization: Bearer $TOKEN"
ja "  berisi tarif_default" '[.data[].key_configs] | contains(["tarif_default"])'
req "GET config by key" GET /penginapan/configs/key/tarif_default 200 -H "Authorization: Bearer $TOKEN"
req "PUT config" PUT "/penginapan/configs/$CONFIG_ID" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"value_configs":"150000"}'
ja "  value terupdate" '.data.value_configs == "150000"'

echo "===== C. LOOKUPS ====="
req "GET payment-methods (seed)" GET /penginapan/payment-methods 200 -H "Authorization: Bearer $TOKEN"
ja "  ada cash & qris" '[.data[].code] | contains(["cash","qris"])'
CASH_ID=$(jget '[.data[] | select(.code=="cash") | .id][0]')
QRIS_ID=$(jget '[.data[] | select(.code=="qris") | .id][0]')
req "GET purchase-sources (seed)" GET /penginapan/purchase-sources 200 -H "Authorization: Bearer $TOKEN"
ja "  ada langsung & mesin" '[.data[].code] | contains(["langsung","mesin"])'

echo "===== D. MASTER ====="
req "POST location -> 201" POST /penginapan/locations 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Lantai 2","keterangan":"Lantai 2"}'
LOC_ID=$(jget .data.id)
req "POST room 201 cap2 -> 201" POST /penginapan/rooms 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_number\":\"201\",\"room_type\":\"pria\",\"capacity\":2}"
R1=$(jget .data.id)
req "POST room 202 cap2 -> 201" POST /penginapan/rooms 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_number\":\"202\",\"room_type\":\"pria\",\"capacity\":2}"
R2=$(jget .data.id)
req "POST room 203 cap1 -> 201" POST /penginapan/rooms 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_number\":\"203\",\"room_type\":\"wanita\",\"capacity\":1}"
R3=$(jget .data.id)
req "POST room 204 cap2 -> 201" POST /penginapan/rooms 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_number\":\"204\",\"room_type\":\"pria\",\"capacity\":2}"
R4=$(jget .data.id)
req "nomor kamar duplikat -> 500 (unique)" POST /penginapan/rooms 500 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_number\":\"201\",\"room_type\":\"pria\"}"
req "POST room-layout -> 201" POST /penginapan/room-layouts 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"location_id\":$LOC_ID,\"room_id\":$R1,\"row_number\":1,\"urutan\":1}"
req "GET room-layouts?location_id" GET "/penginapan/room-layouts?location_id=$LOC_ID" 200 -H "Authorization: Bearer $TOKEN"
ja "  layout ter-load + room" '.data[0].room.room_number == "201"'

echo "===== E. CHECK-IN ====="
req "check-in 1 orang cash lunas -> 201" POST /penginapan/guests 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R1,\"name\":\"Budi\",\"nik\":\"3175\",\"gender\":\"pria\",\"harga\":100000}],\"payment\":{\"amount\":100000,\"payment_method_id\":$CASH_ID,\"source_id\":1}}"
G1=$(jget .data.guests[0].id)
ja "  lunas=true total=100000" '.data.guests[0].lunas == true and .data.guests[0].total_bayar == 100000'
ja "  payment lunas + source langsung" '.data.payment.status == "lunas" and .data.payment.source.code == "langsung" and .data.payment.paid_at != null'
req "check-in 2 orang pending mesin -> 201" POST /penginapan/guests 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R2,\"name\":\"Dewi\",\"gender\":\"wanita\",\"harga\":90000},{\"room_id\":$R2,\"name\":\"Rina\",\"gender\":\"wanita\",\"harga\":90000}],\"payment\":{\"amount\":180000,\"payment_method_id\":$QRIS_ID,\"source_id\":2,\"status\":\"pending\",\"allocations\":[{\"guest_index\":0,\"amount\":90000},{\"guest_index\":1,\"amount\":90000}]}}"
G2A=$(jget .data.guests[0].id)
G2B=$(jget .data.guests[1].id)
PAY_C2=$(jget .data.payment.id)
ja "  pending tidak dihitung: sisa=90000 lunas=false" '.data.guests[0].sisa == 90000 and .data.guests[0].lunas == false and .data.payment.status == "pending" and .data.payment.source.code == "mesin"'
req "multi tanpa allocations -> 400" POST /penginapan/guests 400 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R4,\"name\":\"A\",\"gender\":\"pria\",\"harga\":1},{\"room_id\":$R4,\"name\":\"B\",\"gender\":\"pria\",\"harga\":1}],\"payment\":{\"amount\":2,\"payment_method_id\":$CASH_ID,\"source_id\":1}}"
ja "  pesan allocations" '.message | contains("allocations")'
req "total alloc != amount -> 400" POST /penginapan/guests 400 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R4,\"name\":\"A\",\"gender\":\"pria\",\"harga\":1},{\"room_id\":$R4,\"name\":\"B\",\"gender\":\"pria\",\"harga\":1}],\"payment\":{\"amount\":999,\"payment_method_id\":$CASH_ID,\"source_id\":1,\"allocations\":[{\"guest_index\":0,\"amount\":1},{\"guest_index\":1,\"amount\":1}]}}"
req "melebihi kapasitas (3 di kap1) -> 409" POST /penginapan/guests 409 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R3,\"name\":\"X\",\"gender\":\"wanita\",\"harga\":1},{\"room_id\":$R3,\"name\":\"Y\",\"gender\":\"wanita\",\"harga\":1}]}"
req "room maintenance -> 409" PUT "/penginapan/rooms/$R3/status" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"maintenance"}'
req "check-in ke maintenance -> 409" POST /penginapan/guests 409 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R3,\"name\":\"Z\",\"gender\":\"wanita\",\"harga\":1}]}"
req "GET rooms: occupancy R1=1 terisi" GET /penginapan/rooms 200 -H "Authorization: Bearer $TOKEN"
ja "  R1 occupant 1" ".data[] | select(.id==$R1) | .current_occupants == 1 and .status == \"terisi\""
ja "  R2 occupant 2" ".data[] | select(.id==$R2) | .current_occupants == 2"

echo "===== F. TAMU & PEMBAYARAN ====="
req "GET guests inhouse -> 3" GET /penginapan/guests?status=inhouse 200 -H "Authorization: Bearer $TOKEN"
ja "  jumlah 3 + summary" '(.data | length) == 3 and (.data[0] | has("lunas"))'
req "GET guest G1 lunas" GET "/penginapan/guests/$G1" 200 -H "Authorization: Bearer $TOKEN"
ja "  lunas true" '.data.lunas == true and .data.room.room_number == "201"'
req "PUT guest harga 120000 -> lunas false" PUT "/penginapan/guests/$G1" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"harga":120000}'
req "GET G1 setelah update" GET "/penginapan/guests/$G1" 200 -H "Authorization: Bearer $TOKEN"
ja "  sisa 20000 lunas false" '.data.sisa == 20000 and .data.lunas == false'
req "POST payment partial 50000 -> 201" POST /penginapan/payments 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"amount\":50000,\"payment_method_id\":$CASH_ID,\"source_id\":1,\"guest_ids\":[$G1],\"note\":\"DP\"}"
PMAN=$(jget .data.id)
ja "  alokasi otomatis 1 tamu" '.data.guests[0].amount == 50000 and .data.guests[0].guest_id == '"$G1"
req "GET G1 total 150000 lunas" GET "/penginapan/guests/$G1" 200 -H "Authorization: Bearer $TOKEN"
ja "  total 150000 lunas true" '.data.total_bayar == 150000 and .data.lunas == true'
req "PUT payment pending C2 -> lunas" PUT "/penginapan/payments/$PAY_C2" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"lunas"}'
req "GET G2A lunas setelah C2 lunas" GET "/penginapan/guests/$G2A" 200 -H "Authorization: Bearer $TOKEN"
ja "  lunas true" '.data.lunas == true'
req "GET payments?guest_id=G2A" GET "/penginapan/payments?guest_id=$G2A" 200 -H "Authorization: Bearer $TOKEN"
ja "  1 payment" '(.data | length) == 1 and .data[0].source.code == "mesin"'
req "GET payments?source_id=1 (langsung)" GET "/penginapan/payments?source_id=1" 200 -H "Authorization: Bearer $TOKEN"
ja "  semua source_id=1" 'all(.data[]; .source_id == 1)'
req "PUT amount < alokasi -> 409" PUT "/penginapan/payments/$PMAN" 409 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"amount":10}'
req "DELETE (void) payment partial -> 200" DELETE "/penginapan/payments/$PMAN" 200 -H "Authorization: Bearer $TOKEN"
req "GET G1 setelah void -> sisa 20000" GET "/penginapan/guests/$G1" 200 -H "Authorization: Bearer $TOKEN"
ja "  sisa 20000 lunas false" '.data.sisa == 20000 and .data.lunas == false'
req "DELETE payment lagi -> 409" DELETE "/penginapan/payments/$PMAN" 409 -H "Authorization: Bearer $TOKEN"

echo "===== G. CHECKOUT & PINDAH ====="
req "checkout G1 -> 200" POST "/penginapan/guests/$G1/checkout" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{}'
req "checkout lagi -> 409" POST "/penginapan/guests/$G1/checkout" 409 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{}'
req "R1 kosong setelah checkout" GET /penginapan/rooms 200 -H "Authorization: Bearer $TOKEN"
ja "  R1 occupant 0 kosong" ".data[] | select(.id==$R1) | .current_occupants == 0 and .status == \"kosong\""
req "check-in GX ke R1 -> 201" POST /penginapan/guests 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"guests\":[{\"room_id\":$R1,\"name\":\"Pindah Guy\",\"gender\":\"pria\",\"harga\":50000}]}"
GX=$(jget .data.guests[0].id)
req "pindah GX -> R4 -> 200" POST "/penginapan/guests/$GX/pindah" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"room_id\":$R4}"
ja "  room_lama R1 room_baru R4" '.data.room_lama == '"$R1"' and .data.room_baru == '"$R4"
req "pindah ke kamar sama -> 400" POST "/penginapan/guests/$GX/pindah" 400 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"room_id\":$R4}"
req "GET histories GX -> check-in + pindah" GET "/penginapan/guests/$GX/histories" 200 -H "Authorization: Bearer $TOKEN"
ja "  2 history" '(.data | length) == 2 and .data[1].status == "pindah" and (.data[1].note | contains("201"))'
req "GET room-histories?room_id=R4" GET "/penginapan/room-histories?room_id=$R4" 200 -H "Authorization: Bearer $TOKEN"
ja "  minimal 1" '(.data | length) >= 1'

echo "===== H. GUARD DELETE ====="
req "DELETE room terisi -> 409" DELETE "/penginapan/rooms/$R2" 409 -H "Authorization: Bearer $TOKEN"
req "DELETE room R3 (kosong) -> 200" DELETE "/penginapan/rooms/$R3" 200 -H "Authorization: Bearer $TOKEN"
req "DELETE payment-method dipakai -> 409" DELETE "/penginapan/payment-methods/$CASH_ID" 409 -H "Authorization: Bearer $TOKEN"
req "PUT payment-method rename -> 200" PUT "/penginapan/payment-methods/$QRIS_ID" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"QRIS Updated"}'
req "POST purchase-source baru -> 201" POST /penginapan/purchase-sources 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"code":"online","name":"Online"}'
SRC_ONLINE=$(jget .data.id)
req "DELETE purchase-source dipakai -> 409" DELETE "/penginapan/purchase-sources/1" 409 -H "Authorization: Bearer $TOKEN"
req "DELETE purchase-source baru -> 200" DELETE "/penginapan/purchase-sources/$SRC_ONLINE" 200 -H "Authorization: Bearer $TOKEN"

echo "===== I. STRUK & QRIS ====="
req "GET struk payment lunas (public) -> 200" GET "/penginapan/payments/$PAY_C2/receipt" 200
case "$BODY" in
  *"STRUK PEMBAYARAN"*"Dewi"*) PASS=$((PASS+1)); echo "PASS  struk berisi judul + nama tamu" ;;
  *) FAIL=$((FAIL+1)); echo "FAIL  struk body=${BODY:0:220}" ;;
esac
req "GET struk ?autoprint=1 -> 200" GET "/penginapan/payments/$PAY_C2/receipt?autoprint=1" 200
case "$BODY" in
  *window.print*) PASS=$((PASS+1)); echo "PASS  script autoprint ada" ;;
  *) FAIL=$((FAIL+1)); echo "FAIL  script autoprint hilang body=${BODY:0:220}" ;;
esac
req "GET struk id tak ada -> 404" GET "/penginapan/payments/999999/receipt" 404
req "GET qris tanpa konfigurasi -> 404" GET /penginapan/qris 404 -H "Authorization: Bearer $TOKEN"
req "GET qris tanpa token -> 401" GET /penginapan/qris 401
req "POST config qris_payload -> 201" POST /penginapan/configs 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"key_configs":"qris_payload","value_configs":"00020101021126610014COM.GO-JEK.WWW01189360091432506719960210G1234567890303UMI51440014ID.CO.QRIS.WWW0215ID1020012345678900303UMI5204581253033605802ID5908TTPG PGN6007JAKARTA6105121106304ABCD"}'
req "GET qris -> 200" GET /penginapan/qris 200 -H "Authorization: Bearer $TOKEN"
ja "  payload EMV QRIS" '.data.payload | startswith("000201")'

echo ""
echo "================================"
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
echo "================================"
[ "$FAIL" = "0" ]
