# Testing (smoke test lokal)

Smoke test end-to-end memakai Docker MySQL + MongoDB lokal, bukan DB `192.168.1.3`.

## Prasyarat

```bash
docker run -d --name ttpg-mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=ttpg -p 13306:3306 mysql:8.0
docker run -d --name ttpg-mongo -p 27017:27017 mongo:7

# tunggu MySQL siap, lalu:
docker exec -i ttpg-mysql mysql -uroot -proot -h127.0.0.1 -e \
  "CREATE DATABASE IF NOT EXISTS ttpg_auth;
   CREATE DATABASE IF NOT EXISTS ttpg_penginapan;
   CREATE USER IF NOT EXISTS 'api2'@'%' IDENTIFIED BY 'andtho1989';
   GRANT ALL ON ttpg.* TO 'api2'@'%'; GRANT ALL ON ttpg_auth.* TO 'api2'@'%';
   GRANT ALL ON ttpg_penginapan.* TO 'api2'@'%';"

# skema penginapan (10 tabel + seed payment_methods/purchase_sources):
docker exec -i ttpg-mysql mysql -uroot -proot -h127.0.0.1 ttpg_penginapan \
  < ../../migrations/mysql/002_create_ttpg_penginapan.sql

cd ../..  # root repo
AUTH_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_auth?parseTime=true' go run cmd/seed/main.go
```

## Menjalankan server untuk test

```bash
cd ../..
go build -o /tmp/opencode/ttpg_server .
PORT=8090 \
AUTH_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_auth?parseTime=true' \
MAIN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg?parseTime=true' \
PENGINAPAN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_penginapan?parseTime=true' \
MONGO_URI='mongodb://127.0.0.1:27017' /tmp/opencode/ttpg_server &
```

> `PENGINAPAN_DSN` **wajib** — tanpa itu server memakai default produksi dan `log.Fatalf` saat AutoMigrate bila tidak terjangkau/DB belum ada.

## Skrip

| Skrip | Fungsi |
|-------|--------|
| `reset.sh` | Reset data uji (schema + seed Mongo + row GATE002 + po_ids kosong + re-migrasi `ttpg_penginapan`) |
| `test_api.sh` | 107 assert: auth, gate (valid/rescan/max/expired/fallback), boardingpass (CRUD: list/update/delete + get/print), checkin mandiri, boarding Mongo CRUD, restriksi po_id |
| `test_env_flags.sh` | 5 assert: `BYPASS=true`, `LIST_ID_CARD`, kembali normal (restart server) |
| `test_penginapan.sh` | 77 assert: configs/lookups, master lokasi/kamar/layout, check-in (lunas/pending/alokasi/kapasitas/maintenance), summary lunas, payment manual/void, checkout, pindah kamar, histories, guard delete |
| `test_ws.sh` | 14 assert: auth `/ws` (401 tanpa/salah token), koneksi client, event `room` (status change), event `occupancy` + `guest` (check-in) — build test client otomatis dari `wsclient/` |

```bash
./reset.sh
./test_api.sh         # ekspektasi: TOTAL: PASS=107 FAIL=0
./test_env_flags.sh   # ekspektasi: TOTAL: PASS=5 FAIL=0
./reset.sh && ./test_penginapan.sh   # ekspektasi: TOTAL: PASS=77 FAIL=0
./reset.sh && ./test_ws.sh           # ekspektasi: TOTAL: PASS=14 FAIL=0
```

Catatan:

- Port aplikasi test: **8090** (8080 bisa bentrok dengan proyek lain).
- `test_env_flags.sh` mematikan/menyalakan ulang server via `pkill` — hanya jalankan saat server test (bukan produksi).
- Noreg yang dipakai gate: `SCAN001` (checkin), `MAXSCAN1` (count_scan=5), `EXPIRE01` (time_tab >5 menit), `OLDDATE1` (tanggal kemarin → fallback eksternal), `GATE002`/`T002` (checkin conflict).
- Fallback API tiket (`newticket.dgti.co.id`) dan Damri tidak terjangkau dari sandbox → diverifikasi sebagai jalur gagal (`success:false` / `404`).
- `test_ws.sh` membuat location+kamar unik tiap run (suffix `date +%s`), jadi aman diulang tanpa reset. Jangan pipe output skrip ke `tail` — anak proses server mewarisi stdout sehingga pipe tidak EOF.
