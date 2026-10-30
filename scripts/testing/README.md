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

# skema auth ttpg_users (8 tabel + seed admin/limited; juga dimuat reset.sh):
docker exec -i ttpg-mysql mysql -uroot -proot -h127.0.0.1 < ../../migrations/mysql/004_create_ttpg_users.sql

cd ../..  # root repo
AUTH_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_auth?parseTime=true' go run cmd/seed/main.go
```

## Menjalankan server untuk test

Port **3080** (default skrip test). Build + jalankan dengan env lengkap:

```bash
cd ../..
go build -o ttpg_server .
PORT=3080 \
AUTH_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_auth?parseTime=true' \
MAIN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg?parseTime=true' \
PENGINAPAN_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_penginapan?parseTime=true' \
USERS_DSN='api2:andtho1989@tcp(127.0.0.1:13306)/ttpg_users?parseTime=true' \
MONGO_URI='mongodb://127.0.0.1:27017' ./ttpg_server &
```

> Semua DSN **wajib** lokal — tanpa itu server memakai default produksi (tidak terjangkau dari mesin test) dan `log.Fatalf`. `USERS_DSN` menyediakan login/aturan endpoint (`user`, `access`, `api_url`, `logs`).

## Skrip

| Skrip | Fungsi |
|-------|--------|
| `reset.sh` | Reset data uji (schema + seed Mongo + row GATE002 + po_ids kosong + re-migrasi `ttpg_penginapan` + migrasi/reset tabel vending `006` + `018` + `019` + `020`) |
| `test_api.sh` | 207 assert: auth (JWT/Basic/API key seragam via `ttpg_users`), aturan endpoint (`api_url` wajib Bearer → 403, akses `access.id_api` → 403, blokir user → 403, logs tercatat), boarding Mongo list (paginasi + filter `destination_id`), docs interaktif `/docs`, gate (valid/rescan/max/expired/fallback + scan via JWT/Basic), boardingpass (CRUD: list/update/delete + get/print + secret_key), checkin mandiri, boarding Mongo CRUD, restriksi po_id, activity & logs (halaman /activity public, /activity/logs 401 tanpa token, redaksi password jadi [REDACTED], filter method/path/code/limit, isolasi log per user), vending (config merge server-side + auto-register, heartbeat 30 detik + extra, media urut order, admin media CRUD + upload + purpose screensaver + target screensaver per device / semua) |
| `test_env_flags.sh` | 5 assert: `BYPASS=true`, `LIST_ID_CARD`, kembali normal (restart server) |
| `test_penginapan.sh` | 174 assert: configs/lookups, master lokasi/kamar/layout (termasuk field turunan `available`), check-in (lunas/pending/alokasi/kapasitas/maintenance), summary lunas, foto KTP (upload valid/non-gambar/tanpa file, GET image + content-type, detail `ktp_photo`, GET 404 tanpa foto, delete → null), payment manual/void, checkout, pindah kamar, histories, guard delete, struk pembayaran (public + autoprint + 404), QRIS (404 belum dikonfigurasi → config → 200 payload), payment gateway midtrans/xendit QRIS (mock server lokal `python3` port 18080: charge → qr_string, idempoten, webhook valid/salah signature → lunas, amount mismatch → 403, unknown external_id → 404, EXPIRED → tetap pending, NOT_CONFIGURED), **hold pembayaran cash** (section K: config `payment_hold_minutes=1`, backdate `created_at` via SQL, poll ≤90s → payment batal + auto check-out + kamar available + history note) |
| `test_ws.sh` | 14 assert: auth `/ws` (401 tanpa/salah token), koneksi client, event `room` (status change), event `occupancy` + `guest` (check-in) — build test client otomatis dari `wsclient/` |

```bash
./reset.sh
./test_api.sh         # ekspektasi: TOTAL: PASS=207 FAIL=0
./test_env_flags.sh   # ekspektasi: TOTAL: PASS=5 FAIL=0
./reset.sh && ./test_penginapan.sh   # ekspektasi: TOTAL: PASS=174 FAIL=0
./reset.sh && ./test_ws.sh           # ekspektasi: TOTAL: PASS=14 FAIL=0
```

Catatan:

- Port aplikasi test: **3080** (port permanen; 8080 dipakai proyek lain).
- `test_env_flags.sh` mematikan/menyalakan ulang server via `pkill` — hanya jalankan saat server test (bukan produksi).
- Noreg yang dipakai gate: `SCAN001` (checkin), `MAXSCAN1` (count_scan=5), `EXPIRE01` (time_tab >5 menit), `OLDDATE1` (tanggal kemarin → fallback eksternal), `GATE002`/`T002` (checkin conflict).
- Fallback API tiket (`newticket.dgti.co.id`) dan Damri tidak terjangkau dari sandbox → diverifikasi sebagai jalur gagal (`success:false` / `404`).
- Section J (`test_penginapan.sh`) menjalankan mock server payment gateway dengan `python3` di `127.0.0.1:18080` (dihentikan otomatis via `trap EXIT`) — pastikan `python3` tersedia; API Midtrans/Xendit sungguhan tidak dipanggil.
- `test_ws.sh` membuat location+kamar unik tiap run (suffix `date +%s`), jadi aman diulang tanpa reset. Jangan pipe output skrip ke `tail` — anak proses server mewarisi stdout sehingga pipe tidak EOF. Jangan jalankan `pkill -f` dengan pola yang ada di command-line shell sendiri (bisa match diri sendiri).
