# TTPG — Terminal Terpadu Pulogebang API

API Go (Gin) untuk gate, boardingpass, checkin mandiri, master tujuan/PO, manajemen boarding berbasis MongoDB, dan modul penginapan (kamar, tamu, pembayaran). Menggunakan empat database MySQL (`ttpg_users`, `ttpg_auth`, `ttpg`, `ttpg_penginapan`) dan MongoDB (`ttpg`).

## Fitur

- **Auth**: login `POST /auth` → JWT (berlaku 1 minggu); seragam di semua route protected — JWT (Bearer), Basic, API key (`user.secret_key` di DB `ttpg_users` atau literal env `API_KEY`); aturan skema+izin per endpoint dari `api_url`/`access`, log async ke `logs` (body ikut dicatat **ter-redaksi**, hapus otomatis sesuai `LOG_RETENTION_DAYS`)
- **Gate**: scan noreg, fallback ke API tiket eksternal, aturan max 5 scan & expired 5 menit
- **Checkin mandiri**: dari MongoDB `boarding`, opsional fallback Damri, auto-generate noreg `CM…`
- **Boardingpass**: list (filter+limit/offset), create/get, update/delete (soft), print (cetak HTML + QR)
- **Master data**: `tbl_tujuan`, `tbl_po`, dokumen `boarding` di MongoDB dengan pembatasan `po_id` per user
- **Penginapan** (`/penginapan/*`): master lokasi/kamar/layout, check-in multi-tamu, pembayaran (1 payment → banyak tamu, alokasi per tamu), pindah kamar, check-out, history, **struk pembayaran** (`GET /penginapan/payments/:id/receipt`, public, HTML 78mm + `?autoprint=1`), **QRIS** (`GET /penginapan/qris` dari config `qris_payload`), **payment gateway Midtrans & Xendit** (provider dipilih via config DB `payment_gateway`; `POST /penginapan/payments/:id/pay` + webhook public; Midtrans via `/v2/charge`, Xendit via `/qr_codes` — auth & signature diverifikasi live terhadap sandbox), **field turunan `available`** di semua response kamar (`true` bila kamar kosong — kemudahan client cek ketersediaan), **foto KTP per tamu** (`POST/GET/DELETE /penginapan/guests/:id/ktp`, multipart ≤5MB JPG/PNG/WebP, disimpan di `$UPLOAD_DIR/ktp`), **hold pembayaran cash** (config DB `payment_hold_minutes`, default 15 menit → payment `pending` cash otomatis batal + tamu belum lunas di-check-out otomatis → kamar `available`) — DB khusus `ttpg_penginapan` (GORM, AutoMigrate)
- **WebSocket** (`GET /ws?token=<JWT>`): push real-time perubahan kamar (create/update/status/occupancy/delete) dan tamu (checkin/checkout/pindah) ke semua client terautentikasi
- **Dokumentasi interaktif** (`GET /docs`, public): halaman HTML dengan parameter, body, contoh respons, dan snippet **cURL / PHP / Python / Node.js / Go** per endpoint — plus fitur "Coba" (request langsung dari browser)
- **Aktivitas & Log** (`GET /activity`, public): halaman riwayat request per user — status, data terkirim (query + body, rahasia disamarkan), IP, durasi; filter method/path/status/tanggal + paginasi; API `GET /activity/logs` (user biasa hanya melihat lognya sendiri, admin semua). Log lama dihapus otomatis (`LOG_RETENTION_DAYS`, default 30 hari)
- **Vending / mesin OTOMAT** (`GET /vending/config`, `POST /vending/device/status`, `GET /vending/media`; auth seragam): bundle config per mesin (global + override, **merge di server**), auto-register `status=pending`, heartbeat 30 detik (`last_seen_at`, `extra`), katalog media attract — DB `ttpg` (`vending_machines`/`vending_configs`/`vending_media`, migration `006`)

## Persyaratan

- Go (lihat `go.mod`)
- MySQL/MariaDB — empat database: `ttpg_users` (**auth: user/access/api_url/logs — sumber login, JWT, API key, aturan endpoint**), `ttpg_auth` (legacy users/api_keys, tidak lagi dipakai auth), `ttpg` (transaksi), `ttpg_penginapan` (modul penginapan)
- MongoDB

## Menjalankan

```bash
# 1. skema auth lokal (sekali saja; SKIP di produksi — tabel ttpg_users sudah ada di sana)
#    (Docker test:) docker exec -i ttpg-mysql mysql -uroot -proot < migrations/mysql/004_create_ttpg_users.sql
#    (atau ikuti scripts/testing/README.md)

# 2. jalankan server (port 3080) — set env DSN bila bukan default produksi
#    (lokal contoh: AUTH_DSN/MAIN_DSN/PENGINAPAN_DSN/USERS_DSN → 127.0.0.1:13306, MONGO_URI → lokal)
go run main.go

# 3. buka dokumentasi interaktif di browser
#    http://localhost:3080/docs
```

Seed auth: `migrations/mysql/004_create_ttpg_users.sql` (DB `ttpg_users`) — `admin`/`admin123` (`secret_key=dev-key-123`, `all_access=y`) dan `limited`/`limited123` (akses terbatas, dipakai test aturan endpoint). `go run cmd/seed/main.go` masih mengisi legacy `ttpg_auth` (`production_key=prod-key-123`, `development_key=dev-key-123`) — tidak dipakai auth lagi.

## Akses dari Komputer Lain

Server sudah listen di **semua interface** (`0.0.0.0`), jadi otomatis bisa diakses dari komputer lain di jaringan yang sama (LAN) tanpa ubah kode.

1. **Cari IP server**:

   ```bash
   # Linux
   hostname -I        # atau: ip addr show
   # Windows
   ipconfig
   ```

   Misal hasilnya `192.168.1.50`.

2. **Buka firewall port 3080** (hanya perlu sekali, sesuai OS):

   ```bash
   # Ubuntu/Debian (ufw)
   sudo ufw allow 3080/tcp

   # RHEL/CentOS/Fedora (firewalld)
   sudo firewall-cmd --permanent --add-port=3080/tcp && sudo firewall-cmd --reload

   # iptables manual
   sudo iptables -A INPUT -p tcp --dport 3080 -j ACCEPT
   ```

   Windows: izinkan port di Windows Defender Firewall → Inbound Rules → New Rule → Port 3080.

3. **Akses dari komputer lain**:

   ```
   http://192.168.1.50:3080/auth
   ```

   Verifikasi cepat dari komputer lain:

   ```bash
   curl -X POST http://192.168.1.50:3080/auth \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}'
   ```

   Harus balas `200` dengan `token`. Jika timeout, firewall/port belum dibuka; jika `connection refused`, server belum jalan atau salah IP.

   WebSocket juga lewat port yang sama: `ws://192.168.1.50:3080/ws?token=<JWT>` (lihat [docs/api.md — WebSocket](docs/api.md#websocket)).

4. **Ubah port** (opsional): set env `PORT` sebelum start, mis. `PORT=9000 go run main.go` — sesuaikan firewall & URL.

5. **Akses via internet (WAN)**: port-forward 3080 di router ke IP server, atau pakai reverse proxy (nginx/caddy) + TLS. Untuk produksi, **wajib** ganti `JWT_SECRET` dan `API_KEY` dari default.

## Konfigurasi (env)

| Env | Default | Keterangan |
|-----|---------|------------|
| `PORT` | `3080` | Port server |
| `AUTH_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg_auth?parseTime=true` | DB legacy (users/api_keys — tidak dipakai auth) |
| `USERS_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg_users?parseTime=true` | **DB auth produksi**: `user`, `access`, `api_url`, `logs` |
| `MAIN_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg?parseTime=true` | DB transaksi (terminal_boardingpass, tbl_tujuan, tbl_po) |
| `PENGINAPAN_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg_penginapan?parseTime=true` | DB penginapan (GORM; AutoMigrate saat start) |
| `UPLOAD_DIR` | `uploads` | Folder upload (foto KTP di `$UPLOAD_DIR/ktp`; relatif terhadap CWD server) |
| `MONGO_URI` | `mongodb://…@192.168.1.3:27017/?authSource=admin&directConnection=true` | MongoDB |
| `MONGO_DATABASE` | `ttpg` | Nama database Mongo |
| `MONGO_BOARDING_COLLECTION` | `boarding` | Koleksi boarding |
| `MONGO_BOARDINGPASS_COLLECTION` | `boardingpass` | Koleksi boardingpass checkin |
| `JWT_SECRET` | `ttpg-secret-key-change-in-production` | Secret JWT |
| `API_KEY` | `12345` | API key literal — superkey untuk semua route protected |
| `BYPASS` | `false` | `true` → semua scan gate lolos |
| `LIST_ID_CARD` | *(kosong)* | Daftar noreg yang selalu lolos, dipisah koma |
| `ENABLE_DAMRI_FALLBACK` | `false` | Aktifkan fallback API Damri di checkin |
| `DAMRI_USERNAME` / `DAMRI_PASSWORD` / `DAMRI_BASE_URL` | *(kosong)* | Kredensial Damri |
| `LOG_RETENTION_DAYS` | `30` | Retensi log `ttpg_users.logs` — dihapus otomatis bila lebih tua (0 = nonaktif) |

## Struktur Proyek

```
main.go                  # entrypoint + wiring DB
cmd/seed/                # seed legacy admin+api key (tttg_auth; tidak dipakai auth)
internal/
  config/                # config.Load() → config.C (global; termasuk UsersDSN)
  db/                    # db.DB (*sql.DB), db.MongoClient, db.PenginapanDB + db.UsersDB (*gorm.DB)
  logger/                # Info/Error/Warning
  model/                 # model.Boardingpass (scan SQL, sql.Null*)
  models/                # GORM: User, APIKey, Tujuan, PO, BoardingDoc + 10 model penginapan + users_ttpg.go (auth ttpg_users)
  pkg/response/          # format standar {success,message,data}
  response/              # GateResponse, BoardingpassResponse
  request/               # struct request (termasuk penginapan: checkin/payment/pindah)
  platform/auth/         # JWT, bcrypt, Basic, AuthMiddleware (seragam: Bearer/Basic/API key) + aturan api_url/access + log async
  platform/database/     # ConnectMySQL, ConnectMongo
  handler/               # gate, boardingpass, checkin_mandiri, print_boardingpass,
                         # penginapan_config, penginapan_master, penginapan_guest, penginapan_payment,
                         # penginapan_receipt, penginapan_gateway, activity, vending, ws
  activity/              # halaman UI /activity (embed HTML: riwayat log request user)
  payment/               # gateway pluggable: Gateway interface + registry; midtrans.go, xendit.go
  client/                # TicketClient, DamriClient
  docs/                  # api.html + docs.go (embed halaman /docs)
  routes/routes.go       # registrasi semua route
scripts/testing/         # smoke test lokal (380 assert) + reset DB uji + wsclient/
migrations/mysql/        # 001_create_ttpg_auth.sql, 002_create_ttpg_penginapan.sql (baru), 003_upgrade_penginapan_v2.sql, 004_create_ttpg_users.sql (skema auth + seed lokal), 005_upgrade_payments_gateway.sql (upgrade kolom gateway payments), 006_create_ttpg_vending.sql (tabel vending: machines/configs/media + seed config global), 007_upgrade_guests_ktp.sql (kolom ktp_photo guests)
docs/api.md              # dokumentasi API lengkap
```

## Autentikasi

Semua route protected (gate, checkin-mandiri, boardingpass, tujuan, po, boarding, penginapan) memakai **satu middleware seragam** (`auth.AuthMiddleware`) — detail di [`docs/api.md`](docs/api.md):

1. **API key** — `X-ttpg-key` / `X-API-Key`: **literal** env `API_KEY` (superkey, `user_id=0`) atau `user.secret_key` di `ttpg_users` (user aktif)
2. **JWT** — `Authorization: Bearer <token>` (dari `POST /auth`, berlaku 1 minggu; user dimuat ulang dari `ttpg_users`)
3. **Basic** — `Authorization: Basic base64(username:password)` (bcrypt `ttpg_users.user`)

Digest **dilepas** (tabel `ttpg_users` tidak punya kolom HA1) — skema tak dikenal → `401` + `WWW-Authenticate: Basic realm="ttpg"`.

> **Catatan pembaruan route auth**: login hanya `POST /auth` — path lama `POST /auth/login` dihapus (404), dan `GET /auth/key` sudah dihapus; API key didapat dari kolom `user.secret_key` (superkey: env `API_KEY`).

Setelah identitas valid, middleware mengecek **aturan endpoint** (bila ada baris `api_url` yang cocok method+path; controller ternormalisasi, `/x/index` == `/x`):
- **skema wajib** baris `api_url.authorization` (`api_key`/`bearer_token`/`basic_auth`/`none`) → salah → `403`;
- **izin**: `user.all_access='y'` lolos; selain itu gabungan `user.allowed` + `access.id_api` (dipisah `;`/`,`) harus memuat id `api_url` → salah `403`;
- gate `is_block`/`is_approved` → `401` (saat login/key) / `403` (saat request).

Rute **tanpa** baris `api_url` menerima skema apa pun. Semua request dicatat **async** ke `ttpg_users.logs` (uri, method, params, ip, response_code, authorized, id_user, po_id, access_by). Gagal autentikasi → `401`; terautentikasi tapi ditolak aturan/akses/blokir → `403`.

Contoh cepat:

```bash
TOKEN=$(curl -s -X POST localhost:3080/auth -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r .data.token)

curl -s localhost:3080/tujuan -H "Authorization: Bearer $TOKEN"   # JWT
curl -s localhost:3080/tujuan -H 'X-ttpg-key: dev-key-123'        # API key user (secret_key)
curl -s localhost:3080/tujuan -H 'X-ttpg-key: 12345'              # superkey env API_KEY
curl -s localhost:3080/tujuan -u admin:admin123                   # Basic
```

Tanpa auth: `POST /auth`, `/boarding/print/:code` (alias lama: `/boardingpass/print/:code`), `/penginapan/payments/:id/receipt` (struk), `/penginapan/webhooks/midtrans`, `/penginapan/webhooks/xendit` (gateway), `/ws` (JWT via query `?token=`), `/docs`, `/activity` (UI log).

## Verifikasi

```bash
go build ./...
go vet ./...
```

Smoke test end-to-end (Docker MySQL+Mongo lokal, 380 assert) ada di [`scripts/testing/`](scripts/testing/README.md):

```bash
cd scripts/testing && ./reset.sh && ./test_api.sh && ./test_env_flags.sh && ./test_penginapan.sh && ./reset.sh && ./test_ws.sh
```

(Reset di tengah wajib — `test_ws.sh` butuh state bersih; lihat `scripts/testing/README.md`.)

## Dokumentasi

- **API**: [`docs/api.md`](docs/api.md) — daftar endpoint lengkap seluruh modul
- **API Penginapan (khusus frontend)**: [`docs/api-penginapan.md`](docs/api-penginapan.md) — ringkas, contoh request/response, aturan & jebakan; dikirim ke agent/pengembang frontend penginapan
- **Instruksi agent/kontributor**: [`AGENTS.md`](AGENTS.md) — wajib dibaca; setiap perubahan kode harus memperbarui README ini, `AGENTS.md`, dan `docs/api.md`.
