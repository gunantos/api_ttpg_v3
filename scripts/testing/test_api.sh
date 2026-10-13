#!/bin/bash
BASE=${BASE:-http://127.0.0.1:8090}
KEY="X-ttpg-key: 12345"
PASS=0; FAIL=0
BODY=""; CODE=""

req() {
  local name="$1" method="$2" path="$3" expect="$4"; shift 4
  BODY=$(curl -s -m 15 -w "\n%{http_code}" -X "$method" "$BASE$path" "$@" 2>/dev/null)
  CODE=$(printf '%s' "$BODY" | tail -n1)
  BODY=$(printf '%s' "$BODY" | sed '$d')
  if [ "$CODE" != "$expect" ]; then
    echo "FAIL  $name -> HTTP $CODE (expect $expect) body=${BODY:0:200}"
    FAIL=$((FAIL+1)); return 1
  fi
  PASS=$((PASS+1)); echo "PASS  $name (HTTP $CODE)"
}

ja() {
  local name="$1" expr="$2"
  local got
  got=$(printf '%s' "$BODY" | jq -r "$expr" 2>/dev/null)
  if [ "$got" != "true" ]; then
    echo "FAIL  $name -> jq '$expr' = '${got}' body=${BODY:0:300}"
    FAIL=$((FAIL+1)); return 1
  fi
  PASS=$((PASS+1)); echo "PASS  $name"
}

sqlq() {
  docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg -N -e "$1" 2>/dev/null | tr -d '\r'
}

echo "===== A. AUTH ====="
req "login salah password -> 401" POST /auth/login 401 -H 'Content-Type: application/json' -d '{"username":"admin","password":"salah"}'
ja "  format error standar" '.success == false'

req "login benar -> 200" POST /auth/login 200 -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'
ja "  ada token+user_id" '.data.token != null and .data.user_id == 1'
TOKEN=$(printf '%s' "$BODY" | jq -r .data.token)

req "auth/key Basic -> 200" GET /auth/key 200 -u admin:admin123
ja "  status true + keys" '.status == true and .data.development_key == "dev-key-123"'
req "auth/key tanpa auth -> 401" GET /auth/key 401

req "tujuan tanpa token -> 401" GET /tujuan 401
req "tujuan dengan Bearer -> 200" GET /tujuan 200 -H "Authorization: Bearer $TOKEN"
ja "  4 baris" '.success == true and (.data | length) == 4'
req "po dengan Bearer -> 200" GET /po 200 -H "Authorization: Bearer $TOKEN"
ja "  3 baris, hanya id_po/nama_po" '.success == true and (.data | length) == 3 and (.data[0] | keys | sort) == ["id_po","nama_po"]'
req "tujuan dgn X-ttpg-key hash -> 200" GET /tujuan 200 -H "X-ttpg-key: dev-key-123"
req "tujuan dgn key salah -> 401" GET /tujuan 401 -H "X-ttpg-key: salah"

echo "===== B. GATE ====="
req "gate tanpa key -> 401" POST /gate 401 -H 'Content-Type: application/json' -d '{"noreg":"SCAN001"}'
req "gate key salah -> 401" POST /gate 401 -H "X-ttpg-key: salah" -H 'Content-Type: application/json' -d '{"noreg":"SCAN001","gate_id":1}'
req "gate SCAN001 checkin -> VALID CHECKIN" POST /gate 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"SCAN001","gate_id":1}'
ja "  success+reason" '.success == true and .reason == "VALID CHECKIN"'
req "gate SCAN001 scan ke-2 -> RESCAN" POST /gate 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"SCAN001","gate_id":1}'
ja "  VALID RESCAN" '.success == true and .reason == "VALID RESCAN"'
req "gate MAXSCAN1 -> ditolak" POST /gate 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"MAXSCAN1","gate_id":1}'
ja "  MAXIMUM SCAN" '.success == false and .reason == "MAXIMUM SCAN TERCAPAI"'
req "gate EXPIRE01 -> expired" POST /gate 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"EXPIRE01","gate_id":1}'
ja "  SCAN SUDAH EXPIRED" '.success == false and .reason == "SCAN SUDAH EXPIRED"'
req "gate noreg tak dikenal -> 200 (API eksternal)" POST /gate 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"HILANG01","gate_id":1}'
ja "  success false (eksternal tak terjangkau di sandbox)" '.success == false'
req "gate noreg kosong -> 400" POST /gate 400 -H "$KEY" -H 'Content-Type: application/json' -d '{"gate_id":1}'

SCAN_COUNT=$(sqlq "SELECT COUNT(*) FROM terminal_boardingpass WHERE noreg='SCAN001' AND fungsi='gatein' AND count_scan=2 AND gate_id=1")
if [ "$SCAN_COUNT" = "1" ]; then PASS=$((PASS+1)); echo "PASS  DB: SCAN001 fungsi=gatein count_scan=2"; else FAIL=$((FAIL+1)); echo "FAIL  DB: SCAN001 -> '$SCAN_COUNT'"; fi

echo "===== C. BOARDINGPASS ====="
BP_ID=$(sqlq "SELECT id FROM terminal_boardingpass WHERE noreg='SCAN001'")
req "GET /boardingpass/\$BP_ID -> 200" GET "/boardingpass/$BP_ID" 200 -H "$KEY"
ja "  noreg SCAN001" '.noreg == "SCAN001"'
req "GET /boardingpass/9999 -> 404" GET /boardingpass/9999 404 -H "$KEY"
req "GET /boardingpass/abc -> 400" GET /boardingpass/abc 400 -H "$KEY"
req "POST /boardingpass tanpa key -> 401" POST /boardingpass 401 -H 'Content-Type: application/json' -d '{"noreg":"X"}'
req "POST /boardingpass noreg tak dikenal -> 404" POST /boardingpass 404 -H "$KEY" -H 'Content-Type: application/json' -d '{"noreg":"TIDAKADA1"}'

echo "===== C2. BOARDINGPASS LIST/UPDATE/DELETE ====="
req "list tanpa key -> 401" GET /boardingpass 401
req "GET /boardingpass -> 200" GET /boardingpass 200 -H "$KEY"
ja "  total>=1 + data array" '.total >= 1 and (.data | length) >= 1 and (.data | type) == "array"'
req "list?noreg=SCAN001 -> 200" GET "/boardingpass?noreg=SCAN001" 200 -H "$KEY"
ja "  hanya SCAN001" '.total == 1 and .data[0].noreg == "SCAN001"'
req "list?noreg=TIDAKADA99 -> 200" GET "/boardingpass?noreg=TIDAKADA99" 200 -H "$KEY"
ja "  total 0" '.total == 0 and (.data | length) == 0'
req "list?fungsi=gatein -> 200" GET "/boardingpass?fungsi=gatein" 200 -H "$KEY"
ja "  total>=1" '.total >= 1 and .data[0].fungsi == "gatein"'
req "list?limit=0 -> 400" GET "/boardingpass?limit=0" 400 -H "$KEY"
req "list?id_tujuan=abc -> 400" GET "/boardingpass?id_tujuan=abc" 400 -H "$KEY"
req "list?offset=-1 -> 400" GET "/boardingpass?offset=-1" 400 -H "$KEY"

BP2_ID=$(sqlq "SELECT id FROM terminal_boardingpass WHERE noreg='GATE002' AND deleted IS NULL")
req "PUT tanpa key -> 401" PUT "/boardingpass/$BP2_ID" 401 -H 'Content-Type: application/json' -d '{"fungsi":"x"}'
req "PUT fungsi+count_scan -> 200" PUT "/boardingpass/$BP2_ID" 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"fungsi":"gateout","count_scan":3}'
ja "  baca ulang fungsi+count_scan" '.id == '"$BP2_ID"' and .fungsi.String == "gateout" and .count_scan.Int64 == 3'
BP2_DB=$(sqlq "SELECT CONCAT(fungsi,'|',count_scan) FROM terminal_boardingpass WHERE id=$BP2_ID")
if [ "$BP2_DB" = "gateout|3" ]; then PASS=$((PASS+1)); echo "PASS  DB: fungsi|count_scan = gateout|3"; else FAIL=$((FAIL+1)); echo "FAIL  DB -> '$BP2_DB'"; fi
req "PUT gate_id non-numeric -> 400" PUT "/boardingpass/$BP2_ID" 400 -H "$KEY" -H 'Content-Type: application/json' -d '{"gate_id":"abc"}'
req "PUT gate_id kosong -> NULL" PUT "/boardingpass/$BP2_ID" 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"gate_id":""}'
BP2_DB=$(sqlq "SELECT gate_id IS NULL FROM terminal_boardingpass WHERE id=$BP2_ID")
if [ "$BP2_DB" = "1" ]; then PASS=$((PASS+1)); echo "PASS  DB: gate_id NULL"; else FAIL=$((FAIL+1)); echo "FAIL  DB gate_id -> '$BP2_DB'"; fi
req "PUT time_tab salah format -> 400" PUT "/boardingpass/$BP2_ID" 400 -H "$KEY" -H 'Content-Type: application/json' -d '{"time_tab":"bukan-tanggal"}'
req "PUT time_tab valid -> 200" PUT "/boardingpass/$BP2_ID" 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"time_tab":"2026-01-02 10:20:30"}'
BP2_DB=$(sqlq "SELECT DATE_FORMAT(time_tab,'%Y-%m-%d %H:%i:%s') FROM terminal_boardingpass WHERE id=$BP2_ID")
if [ "$BP2_DB" = "2026-01-02 10:20:30" ]; then PASS=$((PASS+1)); echo "PASS  DB: time_tab = 2026-01-02 10:20:30"; else FAIL=$((FAIL+1)); echo "FAIL  DB time_tab -> '$BP2_DB'"; fi
req "PUT count_scan negatif -> 400" PUT "/boardingpass/$BP2_ID" 400 -H "$KEY" -H 'Content-Type: application/json' -d '{"count_scan":-1}'
req "PUT body kosong -> 400" PUT "/boardingpass/$BP2_ID" 400 -H "$KEY" -H 'Content-Type: application/json' -d '{}'
req "PUT id non-numerik -> 400" PUT /boardingpass/abc 400 -H "$KEY" -H 'Content-Type: application/json' -d '{"fungsi":"x"}'
req "PUT id tidak ada -> 404" PUT /boardingpass/99999 404 -H "$KEY" -H 'Content-Type: application/json' -d '{"fungsi":"x"}'

sqlq "DELETE FROM terminal_boardingpass WHERE noreg='DEL001'" > /dev/null
sqlq "INSERT INTO terminal_boardingpass (noreg,id_po,id_tujuan,nopol,tanggal_berangkat,jam_berangkat,nama_pnp,fungsi,retribusi,created) VALUES ('DEL001','PO001',1,'B 3333 XY',CURDATE(),'10:00:00','Uji Hapus','checkin',0,NOW())" > /dev/null
DEL_ID=$(sqlq "SELECT id FROM terminal_boardingpass WHERE noreg='DEL001'")
req "DELETE tanpa key -> 401" DELETE "/boardingpass/$DEL_ID" 401
req "DELETE /boardingpass/\$DEL_ID -> 200" DELETE "/boardingpass/$DEL_ID" 200 -H "$KEY"
ja "  ada message" '.message == "boardingpass berhasil dihapus"'
req "GET setelah delete -> 404" GET "/boardingpass/$DEL_ID" 404 -H "$KEY"
req "DELETE kedua kalinya -> 404" DELETE "/boardingpass/$DEL_ID" 404 -H "$KEY"
req "DELETE id non-numerik -> 400" DELETE /boardingpass/abc 400 -H "$KEY"
req "list?noreg=DEL001 (soft delete) -> 200" GET "/boardingpass?noreg=DEL001" 200 -H "$KEY"
ja "  tidak muncul di list" '.total == 0'

echo "===== D. PRINT ====="
req "print by noreg -> 200 HTML" GET /boardingpass/print/SCAN001 200
case "$BODY" in *BOARDINGPASS*) PASS=$((PASS+1)); echo "PASS  print body contains BOARDINGPASS";; *) FAIL=$((FAIL+1)); echo "FAIL  print body: ${BODY:0:150}";; esac
case "$BODY" in *"Budi Santoso"*) PASS=$((PASS+1)); echo "PASS  print contains nama dari DB";; *) FAIL=$((FAIL+1)); echo "FAIL  nama tidak ada di print";; esac
req "print by ticket_id -> 200" GET /boardingpass/print/TKT-001 200
req "print tidak dikenal -> 404" GET /boardingpass/print/TIDAK-ADA 404
req "print via query params -> 200 (tanpa DB)" GET "/boardingpass/print/ANYCODE?nama_pnp=Budi&nama_po=PO1&tujuan=Bekasi&tanggal_berangkat=2026-10-12&jam_berangkat=08:00:00&autoprint=1" 200

echo "===== E. CHECKIN MANDIRI ====="
req "checkin tanpa key -> 401" POST /checkin-mandiri 401 -H 'Content-Type: application/json' -d '{"ticket_id":"T001"}'
req "checkin T001 (Mongo) -> 201" POST /checkin-mandiri 201 -H "$KEY" -H 'Content-Type: application/json' -d '{"ticket_id":"T001"}'
ja "  status/source/noreg CM" '.status == true and .source == "mongodb" and (.noreg | startswith("CM")) and (.print_url | contains("/boardingpass/print/"))'
CM_NOREG=$(printf '%s' "$BODY" | jq -r .noreg)
req "checkin T001 ulang -> 200 (sudah checkin)" POST /checkin-mandiri 200 -H "$KEY" -H 'Content-Type: application/json' -d '{"ticket_id":"T001"}'
ja "  data + print_url" '.status == true and (.data.fungsi | if type=="object" then .String else . end) == "checkin"'
req "checkin T002 (sudah gatein hari ini) -> 409" POST /checkin-mandiri 409 -H "$KEY" -H 'Content-Type: application/json' -d '{"ticket_id":"T002"}'
ja "  status false" '.status == false'
req "checkin ticket tak dikenal -> 404" POST /checkin-mandiri 404 -H "$KEY" -H 'Content-Type: application/json' -d '{"ticket_id":"TIDAK-ADA"}'
req "checkin kosong -> 400" POST /checkin-mandiri 400 -H "$KEY" -H 'Content-Type: application/json' -d '{}'

CM_DB=$(sqlq "SELECT COUNT(*) FROM terminal_boardingpass WHERE ticket_id='T001' AND fungsi='checkin' AND noreg LIKE 'CM%'")
if [ "$CM_DB" = "1" ]; then PASS=$((PASS+1)); echo "PASS  DB: row checkin T001 noreg CM ter-insert"; else FAIL=$((FAIL+1)); echo "FAIL  DB: T001 count='$CM_DB'"; fi
CM_MONGO=$(docker exec -i ttpg-mongo mongosh --quiet ttpg --eval 'db.boardingpass.countDocuments({ticket_id:"T001", fungsi:"checkin"})')
if [ "$CM_MONGO" = "1" ]; then PASS=$((PASS+1)); echo "PASS  Mongo: doc boardingpass T001 ter-insert"; else FAIL=$((FAIL+1)); echo "FAIL  Mongo boardingpass T001 = '$CM_MONGO'"; fi
CM_ACTIVE=$(docker exec -i ttpg-mongo mongosh --quiet ttpg --eval 'db.boarding.findOne({ticket_id:"T001"}).status')
if [ "$CM_ACTIVE" = "checkin" ]; then PASS=$((PASS+1)); echo "PASS  Mongo: boarding T001 status=checkin"; else FAIL=$((FAIL+1)); echo "FAIL  Mongo status T001 = '$CM_ACTIVE'"; fi

echo "===== F. BOARDING (Mongo CRUD) ====="
req "POST /boarding -> 201" POST /boarding 201 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"po_id":"PO001","ticket_id":"MB-001","nopol":"B 7777 XY","destination_id":2,"date_of_departure":"2026-10-15","time_of_departure":"07:30:00","name":"Dewi Lestari, Joko Susilo","gender":"P,L"}'
ja "  success + 2 pnp" '.success == true and (.data.pnp | length) == 2'
req "GET /boarding -> 200 list" GET /boarding 200 -H "Authorization: Bearer $TOKEN"
ja "  ada MB-001" '.success == true and ([.data[].ticket_id] | index("MB-001") != null)'
req "GET /boarding filter ticket_id" GET "/boarding?ticket_id=MB-001" 200 -H "Authorization: Bearer $TOKEN"
ja "  tepat 1" '(.data | length) == 1'
BOARD_ID=$(docker exec -i ttpg-mongo mongosh --quiet ttpg --eval 'String(db.boarding.findOne({ticket_id:"MB-001"})._id)')
req "PUT /boarding/:id -> 200" PUT "/boarding/$BOARD_ID" 200 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"status":"inactive","nopol":"B 8888 XY"}'
req "GET setelah PUT (status)" GET "/boarding?ticket_id=MB-001" 200 -H "Authorization: Bearer $TOKEN"
ja "  status inactive" '.data[0].status == "inactive"'
req "PUT id salah format -> 400" PUT "/boarding/bukanhex" 400 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{}'
req "PUT id tak ada -> 404" PUT "/boarding/000000000000000000000000" 404 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{}'
req "DELETE /boarding/:id -> 200" DELETE "/boarding/$BOARD_ID" 200 -H "Authorization: Bearer $TOKEN"
req "GET setelah DELETE -> kosong" GET "/boarding?ticket_id=MB-001" 200 -H "Authorization: Bearer $TOKEN"
ja "  0 doc" '(.data | length) == 0'
req "POST /boarding tanpa nama -> 400" POST /boarding 400 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"ticket_id":"X","destination_id":1,"date_of_departure":"2026-10-15","time_of_departure":"07:00:00"}'

echo "===== G. RESTRIKSI po_id USER ====="
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg_auth -e "UPDATE users SET po_ids='PO001' WHERE username='admin'" 2>/dev/null
req "login lagi (po_ids=PO001)" POST /auth/login 200 -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}'
TOKEN2=$(printf '%s' "$BODY" | jq -r .data.token)
req "POST /boarding po_id PO002 -> 403" POST /boarding 403 -H "Authorization: Bearer $TOKEN2" -H 'Content-Type: application/json' -d '{"po_id":"PO002","ticket_id":"MB-X","destination_id":1,"date_of_departure":"2026-10-15","time_of_departure":"07:00:00","name":"Uji"}'
req "POST /boarding tanpa po_id -> auto-fill PO001" POST /boarding 201 -H "Authorization: Bearer $TOKEN2" -H 'Content-Type: application/json' -d '{"ticket_id":"MB-AUTO","destination_id":1,"date_of_departure":"2026-10-15","time_of_departure":"07:00:00","name":"Uji"}'
AUTO_PO=$(docker exec -i ttpg-mongo mongosh --quiet ttpg --eval 'db.boarding.findOne({ticket_id:"MB-AUTO"}).po_id')
if [ "$AUTO_PO" = "PO001" ]; then PASS=$((PASS+1)); echo "PASS  auto-fill po_id=PO001"; else FAIL=$((FAIL+1)); echo "FAIL  po_id='$AUTO_PO'"; fi
req "GET /boarding terfilter hanya PO001" GET /boarding 200 -H "Authorization: Bearer $TOKEN2"
ja "  tidak ada PO002" '([.data[].po_id] | contains(["PO002"])) | not'
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg_auth -e "UPDATE users SET po_ids='' WHERE username='admin'" 2>/dev/null

echo ""
echo "================================"
echo "TOTAL: PASS=$PASS FAIL=$FAIL"
echo "================================"
[ "$FAIL" = "0" ]
