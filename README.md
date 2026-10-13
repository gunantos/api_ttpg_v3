# TTPG — Terminal Terpadu Pulogebang API

API Go (Gin) untuk gate, boardingpass, checkin mandiri, master tujuan/PO, manajemen boarding berbasis MongoDB, dan modul penginapan (kamar, tamu, pembayaran). Menggunakan tiga database MySQL (`ttpg_auth`, `ttpg`, `ttpg_penginapan`) dan MongoDB (`ttpg`).

## Fitur

- **Auth**: JWT (login) + API key (tabel `api_keys`) + Basic/Digest (`GET /auth/key`)
- **Gate**: scan noreg, fallback ke API tiket eksternal, aturan max 5 scan & expired 5 menit
- **Checkin mandiri**: dari MongoDB `boarding`, opsional fallback Damri, auto-generate noreg `CM…`
- **Boardingpass**: list (filter+limit/offset), create/get, update/delete (soft), print (cetak HTML + QR)
- **Master data**: `tbl_tujuan`, `tbl_po`, dokumen `boarding` di MongoDB dengan pembatasan `po_id` per user
- **Penginapan** (`/penginapan/*`): master lokasi/kamar/layout, check-in multi-tamu, pembayaran (1 payment → banyak tamu, alokasi per tamu), pindah kamar, check-out, history — DB khusus `ttpg_penginapan` (GORM, AutoMigrate)
- **WebSocket** (`GET /ws?token=<JWT>`): push real-time perubahan kamar (create/update/status/occupancy/delete) dan tamu (checkin/checkout/pindah) ke semua client terautentikasi

## Persyaratan

- Go (lihat `go.mod`)
- MySQL/MariaDB — tiga database: `ttpg_auth` (users/api_keys), `ttpg` (transaksi), `ttpg_penginapan` (modul penginapan)
- MongoDB

## Menjalankan

```bash
# 1. seed admin + api key (sekali saja)
go run cmd/seed/main.go

# 2. jalankan server (port 8080)
go run main.go
```

Seed menghasilkan: `admin` / `admin123`, `production_key=prod-key-123`, `development_key=dev-key-123`, `X-ttpg-key: dev-key-123`.

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

2. **Buka firewall port 8080** (hanya perlu sekali, sesuai OS):

   ```bash
   # Ubuntu/Debian (ufw)
   sudo ufw allow 8080/tcp

   # RHEL/CentOS/Fedora (firewalld)
   sudo firewall-cmd --permanent --add-port=8080/tcp && sudo firewall-cmd --reload

   # iptables manual
   sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
   ```

   Windows: izinkan port di Windows Defender Firewall → Inbound Rules → New Rule → Port 8080.

3. **Akses dari komputer lain**:

   ```
   http://192.168.1.50:8080/auth/login
   ```

   Verifikasi cepat dari komputer lain:

   ```bash
   curl -X POST http://192.168.1.50:8080/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}'
   ```

   Harus balas `200` dengan `token`. Jika timeout, firewall/port belum dibuka; jika `connection refused`, server belum jalan atau salah IP.

   WebSocket juga lewat port yang sama: `ws://192.168.1.50:8080/ws?token=<JWT>` (lihat [docs/api.md — WebSocket](docs/api.md#websocket)).

4. **Ubah port** (opsional): set env `PORT` sebelum start, mis. `PORT=9000 go run main.go` — sesuaikan firewall & URL.

5. **Akses via internet (WAN)**: port-forward 8080 di router ke IP server, atau pakai reverse proxy (nginx/caddy) + TLS. Untuk produksi, **wajib** ganti `JWT_SECRET` dan `API_KEY` dari default.

## Konfigurasi (env)

| Env | Default | Keterangan |
|-----|---------|------------|
| `PORT` | `8080` | Port server |
| `AUTH_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg_auth?parseTime=true` | DB auth (users, api_keys) |
| `MAIN_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg?parseTime=true` | DB transaksi (terminal_boardingpass, tbl_tujuan, tbl_po) |
| `PENGINAPAN_DSN` | `api2:…@tcp(192.168.1.3:3306)/ttpg_penginapan?parseTime=true` | DB penginapan (GORM; AutoMigrate saat start) |
| `MONGO_URI` | `mongodb://…@192.168.1.3:27017/?authSource=admin` | MongoDB |
| `MONGO_DATABASE` | `ttpg` | Nama database Mongo |
| `MONGO_BOARDING_COLLECTION` | `boarding` | Koleksi boarding |
| `MONGO_BOARDINGPASS_COLLECTION` | `boardingpass` | Koleksi boardingpass checkin |
| `JWT_SECRET` | `ttpg-secret-key-change-in-production` | Secret JWT |
| `API_KEY` | `12345` | API key literal untuk group gate/checkin/boardingpass |
| `BYPASS` | `false` | `true` → semua scan gate lolos |
| `LIST_ID_CARD` | *(kosong)* | Daftar noreg yang selalu lolos, dipisah koma |
| `ENABLE_DAMRI_FALLBACK` | `false` | Aktifkan fallback API Damri di checkin |
| `DAMRI_USERNAME` / `DAMRI_PASSWORD` / `DAMRI_BASE_URL` | *(kosong)* | Kredensial Damri |

## Struktur Proyek

```
main.go                  # entrypoint + wiring DB
cmd/seed/                # seed admin & api key
internal/
  config/                # config.Load() → config.C (global)
  db/                    # db.DB (*sql.DB), db.MongoClient, db.PenginapanDB (*gorm.DB)
  middleware/apikey.go    # ApiKeyAuth (header X-ttpg-key / X-API-Key)
  model/                 # model.Boardingpass (scan SQL, sql.Null*)
  models/                # GORM: User, APIKey, Tujuan, PO, BoardingDoc + 10 model penginapan
  pkg/response/          # format standar {success,message,data}
  response/              # GateResponse, BoardingpassResponse
  request/               # struct request (termasuk penginapan: checkin/payment/pindah)
  platform/auth/         # JWT, bcrypt, Basic/Digest, AuthMiddleware
  platform/database/     # ConnectMySQL, ConnectMongo
  handler/               # gate, boardingpass, checkin_mandiri, print_boardingpass,
                         # penginapan_config, penginapan_master, penginapan_guest, penginapan_payment, ws
  client/                # TicketClient, DamriClient
  routes/routes.go       # registrasi semua route
scripts/testing/         # smoke test lokal (203 assert) + reset DB uji + wsclient/
migrations/mysql/        # 001_create_ttpg_auth.sql, 002_create_ttpg_penginapan.sql (baru), 003_upgrade_penginapan_v2.sql (upgrade skema lama)
docs/api.md              # dokumentasi API lengkap
```

## Autentikasi

Tiga mekanisme, tergantung kelompok route (detail di [`docs/api.md`](docs/api.md)):

1. **API key literal** (`X-ttpg-key` / `X-API-Key` = env `API_KEY`) → `/gate`, `/checkin-mandiri`, `/boardingpass*`
2. **JWT** (`Authorization: Bearer <token>`, berlaku 1 minggu) atau **API key ter-hash** → `/tujuan`, `/po`, `/boarding*`, `/penginapan/*`
3. **Basic / Digest** → `GET /auth/key`
4. `GET /boardingpass/print/:code` → **tanpa auth**

## Verifikasi

```bash
go build ./...
go vet ./...
```

Smoke test end-to-end (Docker MySQL+Mongo lokal, 203 assert) ada di [`scripts/testing/`](scripts/testing/README.md):

```bash
cd scripts/testing && ./reset.sh && ./test_api.sh && ./test_env_flags.sh && ./test_penginapan.sh && ./test_ws.sh
```

## Dokumentasi

- **API**: [`docs/api.md`](docs/api.md)
- **Instruksi agent/kontributor**: [`AGENTS.md`](AGENTS.md) — wajib dibaca; setiap perubahan kode harus memperbarui README ini, `AGENTS.md`, dan `docs/api.md`.
