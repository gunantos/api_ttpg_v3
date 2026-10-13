# AGENTS.md — Instruksi untuk Agent

**PENTING: Setiap kali mengubah kode (tambah/ubah/hapus endpoint, handler, middleware, config, model, struktur), WAJIB memperbarui `README.md`, `docs/api.md`, dan file ini. Dokumentasi tidak boleh tertinggal dari kode.**

## Verifikasi (wajib dijalankan setelah setiap perubahan)

```bash
go build ./...
go vet ./...
```

Smoke test end-to-end (Docker MySQL+Mongo lokal, 203 assert) ada di `scripts/testing/` — jalankan `reset.sh`, `test_api.sh`, `test_env_flags.sh`, `test_penginapan.sh`, `test_ws.sh` (lihat `scripts/testing/README.md`). Jangan jalankan skrip test terhadap DB produksi (`192.168.1.3`).

## Lingkungan

- Module: `github.com/andtho89/ttpg`, Go (lihat `go.mod`)
- Server berjalan: `go run main.go` (port default `8080`)
- Seed awal: `go run cmd/seed/main.go`
- **Server WAJIB dapat ketiga-nya**: `AUTH_DSN`, `MAIN_DSN`, `PENGINAPAN_DSN` (default-nya menunjuk DB produksi; jika DB `ttpg_penginapan` belum ada atau user ditolak, server gagal start — `log.Fatalf` saat AutoMigrate). Untuk test lokal Docker: `PENGINAPAN_DSN` → `127.0.0.1:13306` (lihat `scripts/testing/test_env_flags.sh` / `start` pattern di skrip).
- **Sandbox TIDAK bisa mencapai `192.168.1.3:3306` (MySQL) dan `27017` (Mongo)** — test runtime harus dijalankan di mesin user atau Docker lokal. Jangan mengubah default DSN/URI hanya karena koneksi gagal di sandbox.

## Struktur

```
main.go                     # entrypoint: config, connect 3 MySQL + Mongo, AutoMigrate penginapan, routes
cmd/seed/                   # seed user admin + api key (tttg_auth)
internal/
  config/                   # config.Load() → global config.C (env-driven; termasuk PenginapanDSN)
  db/                       # global db.DB (*sql.DB), db.MongoClient, db.PenginapanDB (*gorm.DB)
  logger/                   # Info/Error/Warning (fmt-based)
  middleware/apikey.go       # ApiKeyAuth(config.C.APIKey) → header X-ttpg-key / X-API-Key
  model/boardingpass.go      # model.Boardingpass (sql.Null* untuk kolom nullable)
  models/                    # GORM: User, APIKey, Tujuan, PO, BoardingDoc + penginapan.go (10 model: configs, payment_methods, purchase_sources, locations, rooms, guests, payments, payment_guests, room_histories, room_layouts)
  pkg/response/              # format standar {success,message,data,pagination,errors}
  response/                  # format gate/boardingpass: GateResponse, BoardingpassResponse
  request/                   # struct request: Gate, Create/Update Boardingpass, Checkin + penginapan.go (checkin/payment/pindah/checkout)
  platform/auth/             # JWT, bcrypt login, Basic/Digest, AuthMiddleware
  platform/database/         # ConnectMySQL, ConnectMongo
  handler/                   # gate, boardingpass, checkin_mandiri, print_boardingpass,
                             # penginapan_config, penginapan_master, penginapan_guest, penginapan_payment, ws
                             # (ws.go: hub WebSocket + broadcast event kamar/tamu)
  client/                    # TicketClient (newticket.dgti.co.id), DamriClient
  routes/routes.go           # semua registrasi route
  repositories/, services/   # (kosong, placeholder)
scripts/testing/             # smoke test lokal: reset.sh, test_api.sh, test_env_flags.sh, test_penginapan.sh, test_ws.sh + wsclient/ (jangan arahkan ke DB produksi)
migrations/mysql/            # 001_create_ttpg_auth.sql, 002_create_ttpg_penginapan.sql (skema canonical penginapan + seed), 003_upgrade_penginapan_v2.sql (upgrade skema lama produksi → skema baru, tanpa DROP data)
```

## Konvensi

- Framework HTTP: **Gin**. Handler bernama `func(c *gin.Context)` di `internal/handler/` atau closure di `routes.go`.
- **Dua format response, jangan disamakan:**
  1. `internal/pkg/response` (`response.Success` / `response.Error`) → untuk `/auth/login`, `/tujuan`, `/po`, `/boarding*`, **seluruh `/penginapan/*`**. Shape: `{success, message, data, pagination?, errors?}`.
  2. Format mentah handler gate/boardingpass/checkin (`gin.H` atau struct `internal/response`) → untuk `/gate`, `/checkin-mandiri`, `/boardingpass*`, `/boardingpass/print`. Jangan dibungkus format standar.
  3. `/auth/key` memakai shape sendiri: `{status, data, message}`.
- Akses DB:
  - Query mentah `db.DB` (*sql.DB) untuk `terminal_boardingpass` dan `tbl_tujuan`.
  - GORM via `mainDB` (parameter `routes.RegisterRoutes`) untuk `tbl_po`, `/tujuan`.
  - `auth.AuthDB` (GORM) untuk `users`/`api_keys`.
  - `db.MongoClient.Database(config.C.MongoDatabase).Collection(...)` untuk Mongo.
  - `db.PenginapanDB` (GORM, env `PENGINAPAN_DSN`) untuk semua tabel penginapan; model + AutoMigrate ada di `main.go`.
- Kolom nullable di-`scan` ke field `sql.Null*` pada `model.Boardingpass` (**bukan** pointer) — scan ke `**T` akan error.
- **Kolom MySQL `TIME` di model penginapan harus `*string`** (`checkin_time`/`checkout_time`), **bukan** `*time.Time` — driver mengembalikan `[]byte`/string untuk TIME dan scan ke `*time.Time` gagal. Format `HH:MM:SS` (bangun dengan `fmt.Sprintf("%02d:%02d:%02d", …)`).
- **Tag GORM untuk kolom TIME WAJIB `type:time(0)`** — `type:time` (huruf apa pun) di-lowercase parser GORM jadi `schema.Time` dan AutoMigrate mengubah kolom jadi `datetime(3)` (merusak data!). `type:time(0)` lolos ke custom-type dan tetap `time`, tanpa ALTER berulang.
- Kolom `int` di model penginapan diberi `type:int(11)` eksplisit agar cocok dengan DDL `002_create_ttpg_penginapan.sql` (AutoMigrate default `bigint` → FK incompatible). `row_number` adalah reserved word — backtick di DDL.
- Jangan tambahkan komentar di kode kecuali diminta.
- Tambah dependency: cek dulu sudah dipakai di `go.mod` (`gin`, `gorm`, `jose/jwt` sudah ada).

## Urutan route (jangan diubah sembarangan)

Di `internal/routes/routes.go`:

1. `GET /boardingpass/print/:code` — **public, tanpa auth, harus terdaftar duluan.**
2. `GET /ws` — **public route**; auth dilakukan di handler via query `?token=<JWT>` (401 sebelum upgrade).
3. Group `middleware.ApiKeyAuth(config.C.APIKey)` — literal cocok dengan env `API_KEY`: `POST /gate`, `POST /checkin-mandiri`, `GET /boardingpass` (list), `GET/POST /boardingpass`, `GET/PUT/DELETE /boardingpass/:id`.
4. `POST /auth/login`, `GET /auth/key`.
5. Group `auth.AuthMiddleware()` — `GET /tujuan`, `GET /po`, `GET/POST/PUT/DELETE /boarding*`.
6. Group `peng := r.Group("/penginapan")` + `auth.AuthMiddleware()` — configs, payment-methods, purchase-sources, locations, rooms (+`PUT /rooms/:id/status`), room-layouts, guests (+`/guests/:id/checkout`, `/guests/:id/pindah`, `/guests/:id/histories`), payments, room-histories.

**Dua makna header `X-ttpg-key` berbeda** (jangan disatukan):
- Di group ApiKeyAuth: dibandingkan literal dengan env `API_KEY` (default `12345`).
- Di group AuthMiddleware: di-hash SHA-256 lalu dicocokkan ke tabel `api_keys` (`auth.HashKey`).

## Aturan bisnis (jangan diubah tanpa konfirmasi user)

- **Gate** (`handler/gate.go`): fallback ke API tiket eksternal bila noreg tidak ada (insert `PO122`, `fungsi=gatein`); max 5 scan; rescan kedaluwarsa > 5 menit; `BYPASS=true` → selalu lolos; `LIST_ID_CARD` (env, dipisah koma) → selalu lolos.
- **Checkin mandiri** (`handler/checkin_mandiri.go`): cari di Mongo (`status=active`), jika kosong dan `ENABLE_DAMRI_FALLBACK=true` baru coba Damri; rollback Mongo bila MySQL gagal; noreg = `"CM" + hex(16 byte)`; PO `PO028` = Damri.
- **po_id user**: disimpan sebagai string dipisah koma di `users.po_ids`, di-parse jadi `[]string`; 1 po_id → auto-fill; 0 → bebas.
- **Penginapan** (`handler/penginapan_*.go`, DB `ttpg_penginapan`):
  - Lunas = Σ alokasi dari payment `status='lunas'` ≥ `harga` tamu; `pending` tidak dihitung.
  - 1 payment bisa menutup banyak tamu: wajib `allocations` (total = `amount`); 1 tamu → alokasi otomatis.
  - `rooms.current_occupants` + `status` disinkronkan dalam transaksi (`adjustRoomOccupancy`); check-in tolak (409) bila kapasitas penuh/maintenance.
  - Pindah kamar: update `room_id` in-place + history `pindah` (note `"lama -> baru"`); tidak ada baris tamu baru.
  - Checkout: isi `checkout_date`/`checkout_time` + history `check-out`; ganda → 409.
  - `DELETE` payment = void → `status='batal'` (409 bila sudah batal); `PUT` payment `amount` < total alokasi → 409.
  - `DELETE` room dengan tamu aktif → 409; `DELETE` payment-method/purchase-source yang dipakai → 409.
  - Broadcast WebSocket (`handler/ws.go`): setelah commit, event `type=room` (action create/update/status/delete + occupancy) dan `type=guest` (checkin/checkout/pindah) dikirim ke semua client `/ws` yang terautentikasi.

## Pekerjaan belum selesai (lanjutkan bila diminta)

- Test runtime sudah dilakukan via `scripts/testing/` (203 assert: 107 API + 5 env-flag + 77 penginapan + 14 WS, Docker lokal) terhadap semua endpoint yang ada; sisa yang belum teruji: jalur sukses fallback API tiket & Damri (butuh akses jaringan keluar).
