# AGENTS.md — Instruksi untuk Agent

**PENTING: Setiap kali mengubah kode (tambah/ubah/hapus endpoint, handler, middleware, config, model, struktur), WAJIB memperbarui `README.md`, `docs/api.md`, dan file ini. Dokumentasi tidak boleh tertinggal dari kode.**

## Verifikasi (wajib dijalankan setelah setiap perubahan)

```bash
go build ./...
go vet ./...
```

Smoke test end-to-end (Docker MySQL+Mongo lokal, 248 assert) ada di `scripts/testing/` — jalankan `reset.sh`, `test_api.sh`, `test_env_flags.sh`, `test_penginapan.sh`, `test_ws.sh` (lihat `scripts/testing/README.md`). Jangan jalankan skrip test terhadap DB produksi (`192.168.1.3`).

## Lingkungan

- Module: `github.com/andtho89/ttpg`, Go (lihat `go.mod`)
- Server berjalan: `go run main.go` (port default `3080`)
- Seed awal: `go run cmd/seed/main.go`
- **Server WAJIB dapat keempat-nya**: `AUTH_DSN`, `MAIN_DSN`, `PENGINAPAN_DSN`, **`USERS_DSN`** (auth — default menunjuk DB produksi `ttpg_users`; jika DB/user ditolak → `log.Fatalf`). Untuk test lokal Docker: semua DSN → `127.0.0.1:13306` (lihat `scripts/testing/test_env_flags.sh` `restart()` / `run_server.sh`).
- **Auth membaca `ttpg_users`** (env `USERS_DSN`): login/JWT/API key/Basic dari tabel `user`; aturan endpoint dari `api_url` + `access`/`allowed`; request dicatat async ke `logs`. Tabel `ttpg_auth.users`/`api_keys` **tidak dipakai auth lagi** (masih di-seed legacy).
- **Sandbox TIDAK bisa mencapai `192.168.1.3:3306` (MySQL) dan `27017` (Mongo)** — test runtime harus dijalankan di mesin user atau Docker lokal. Jangan mengubah default DSN/URI hanya karena koneksi gagal di sandbox.
- **`MONGO_URI` default WAJIB menyertakan `directConnection=true`**: Mongo produksi adalah replica-set `rs0` yang mengiklankan host `localhost.localdomain:27017`; tanpa directConnection driver Go menemukan primary di localhost yang salah → `ReplicaSetNoPrimary, Servers: []` (mongosh tetap jalan — jangan pakai mongosh sebagai patokan driver).

## Struktur

```
main.go                     # entrypoint: config, connect 4 MySQL (auth/main/penginapan/users) + Mongo, AutoMigrate penginapan, routes
cmd/seed/                   # seed legacy admin+api key (tttg_auth; auth asli dari migration 004 ttpg_users)
internal/
  config/                   # config.Load() → global config.C (env-driven; termasuk PenginapanDSN, UsersDSN)
  db/                       # global db.DB (*sql.DB), db.MongoClient, db.PenginapanDB (*gorm.DB), db.UsersDB (*gorm.DB)
  logger/                   # Info/Error/Warning (fmt-based)
  model/boardingpass.go      # model.Boardingpass (sql.Null* untuk kolom nullable)
  models/                    # GORM: User, APIKey, Tujuan, PO, BoardingDoc + penginapan.go (10 model) + users_ttpg.go (TtpgUser/TtpgAccess/TtpgApiURL/TtpgLog → tabel user/access/api_url/logs di ttpg_users)
  pkg/response/              # format standar {success,message,data,pagination,errors}
  response/                  # format gate/boardingpass: GateResponse, BoardingpassResponse
  request/                   # struct request: Gate, Create/Update Boardingpass, Checkin + penginapan.go (checkin/payment/pindah/checkout)
  platform/auth/             # JWT, bcrypt login, Basic, AuthMiddleware (SERAGAM: Bearer/Basic/X-ttpg-key|X-API-Key) + rules.go (aturan api_url/access, log async)
  platform/database/         # ConnectMySQL, ConnectMongo
  handler/                   # gate, boardingpass, checkin_mandiri, print_boardingpass,
                             # penginapan_config, penginapan_master, penginapan_guest, penginapan_payment, penginapan_receipt, ws
                             # (ws.go: hub WebSocket + broadcast event kamar/tamu)
  docs/                      # api.html (docs interaktif /docs, di-embed via docs.go)
  client/                    # TicketClient (newticket.dgti.co.id), DamriClient
  routes/routes.go           # semua registrasi route
  repositories/, services/   # (kosong, placeholder)
scripts/testing/             # smoke test lokal: reset.sh, test_api.sh, test_env_flags.sh, test_penginapan.sh, test_ws.sh + wsclient/ (jangan arahkan ke DB produksi)
migrations/mysql/            # 001_create_ttpg_auth.sql, 002_create_ttpg_penginapan.sql (skema canonical penginapan + seed), 003_upgrade_penginapan_v2.sql (upgrade skema lama, tanpa DROP data), 004_create_ttpg_users.sql (skema auth + seed lokal; dimuat reset.sh, TIDAK untuk produksi)
```

## Konvensi

- Framework HTTP: **Gin**. Handler bernama `func(c *gin.Context)` di `internal/handler/` atau closure di `routes.go`.
- **Dua format response, jangan disamakan:**
  1. `internal/pkg/response` (`response.Success` / `response.Error`) → untuk `/auth/login`, `/tujuan`, `/po`, `/boarding*`, **seluruh `/penginapan/*`**. Shape: `{success, message, data, pagination?, errors?}`. (Satu-satunya pengecualian 200 OK: `GET /penginapan/payments/:id/receipt` mengembalikan HTML struk; error-nya tetap format standar.)
  2. Format mentah handler gate/boardingpass/checkin (`gin.H` atau struct `internal/response`) → untuk `/gate`, `/checkin-mandiri`, `/boardingpass*`, `/boarding/print`. Jangan dibungkus format standar.
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

1. `GET /boarding/print/:code` — **public, tanpa auth, harus terdaftar duluan.** (`GET /boardingpass/print/:code` = alias lama, handler sama; jangan dihapus tanpa konfirmasi — frontend lama masih pakai.)
2. `GET /ws` — **public route**; auth dilakukan di handler via query `?token=<JWT>` (401 sebelum upgrade).
3. `GET /docs` — **public**; halaman dokumentasi interaktif (embed `internal/docs/api.html`; update juga `docs/api.md` bila endpoint berubah).
4. `GET /penginapan/payments/:id/receipt` — **public, tanpa auth** (struk HTML 78mm; dibuka via `window.open`/`?autoprint=1` seperti `/boarding/print`, jadi tidak boleh masuk group JWT).
5. Group `auth.AuthMiddleware()` (seragam — satunya middleware untuk semua route protected; ganti `middleware.ApiKeyAuth` lama): `POST /gate`, `POST /checkin-mandiri`, `GET /boardingpass` (list), `GET/POST /boardingpass`, `GET/PUT/DELETE /boardingpass/:id`.
6. `POST /auth/login`, `GET /auth/key`.
7. Group `auth.AuthMiddleware()` — `GET /tujuan`, `GET /po`, `GET/POST/PUT/DELETE /boarding*`.
8. Group `peng := r.Group("/penginapan")` + `auth.AuthMiddleware()` — configs, **`GET /qris`**, payment-methods, purchase-sources, locations, rooms (+`PUT /rooms/:id/status`), room-layouts, guests (+`/guests/:id/checkout`, `/guests/:id/pindah`, `/guests/:id/histories`), payments, room-histories.

**Auth seragam** (`auth.AuthMiddleware`, berlaku untuk SEMUA route protected — gate/checkin/boardingpass/tujuan/po/boarding/penginapan; sumber data **`ttpg_users`** via env `USERS_DSN`):
- Urutan: header `X-ttpg-key`/`X-API-Key` → **literal** env `API_KEY` (superkey; `user_id=0`, `role=api_key`, `po_ids=[]`) dulu, baru `user.secret_key` (user aktif: `is_block=0`, `is_approved≠0`). Lalu `Authorization`: `Bearer` (JWT; baris `user` dimuat ulang per request) / `Basic` (bcrypt `ttpg_users.user`). **Digest DILEPAS** (tabel user tidak punya HA1) → skema tak dikenal 401.
- Setelah identitas valid → **aturan endpoint** (`rules.go`): cari baris `api_url` cocok method+path (controller ternormalisasi: lowercase, tanpa `/`, `/x/index`==`/x`, dibandingkan juga dengan pola Gin); bila ada: skema wajib `authorization` (api_key/bearer_token/basic_auth/none) → salah **403**; izin `all_access='y'` bypass, selain itu gabungan `user.allowed` + `access.id_api` (split `;`/`,`) harus memuat id `api_url`, `access.block='n'`+`allow='y'` → salah **403**. Tanpa baris `api_url` → skema bebas.
- Gate user: `is_block=1`/`is_approved=0` → 401 saat login/key, 403 saat request. Gagal auth → 401 `{success:false,...}` + header `WWW-Authenticate: Basic realm="ttpg"`; ditolak aturan/akses → 403.
- Request dicatat **async** (channel, jangan blocking) ke `ttpg_users.logs` — uri/method/params/ip/response_code/authorized/id_user/po_id/access_by.
- Package `internal/middleware` sudah DIHAPUS; jangan tambahkan group auth terpisah — cukup `auth.AuthMiddleware()`.

## Aturan bisnis (jangan diubah tanpa konfirmasi user)

- **Gate** (`handler/gate.go`): fallback ke API tiket eksternal bila noreg tidak ada (insert `PO122`, `fungsi=gatein`); max 5 scan; rescan kedaluwarsa > 5 menit; `BYPASS=true` → selalu lolos; `LIST_ID_CARD` (env, dipisah koma) → selalu lolos.
- **Checkin mandiri** (`handler/checkin_mandiri.go`): cari di Mongo (`status=active`), jika kosong dan `ENABLE_DAMRI_FALLBACK=true` baru coba Damri; rollback Mongo bila MySQL gagal; noreg = `"CM" + hex(16 byte)`; PO `PO028` = Damri; `print_url` memakai `/boarding/print/<noreg>` (bukan `/boardingpass/print`).
- **Boarding (Mongo `ttpg.boarding`)**: `destination_id` campur tipe di produksi — int (967k), string `"395"` (260k), object `{id_tujuan: 395, …}` (8,5k), 1 missing → **wajib** `models.FlexInt` (`bson.ValueUnmarshaler`) di `BoardingDoc`/`mongoBoarding`; decode default (int saja) membuat `GET /boarding` 500. Filter `GET /boarding?destination_id=` memakai `$or` [int, string, `destination_id.id_tujuan`]. `GET /boarding` **wajib paginasi** `limit` (1-1000, default 100) + `offset` + `pagination` — tanpa limit handler memuat 1,2M dokumen ke memori (timeout/OOM di produksi).
- **po_id user**: disimpan sebagai string dipisah koma di `users.po_ids`, di-parse jadi `[]string`; 1 po_id → auto-fill; 0 → bebas.
- **Penginapan** (`handler/penginapan_*.go`, DB `ttpg_penginapan`):
  - Lunas = Σ alokasi dari payment `status='lunas'` ≥ `harga` tamu; `pending` tidak dihitung.
  - 1 payment bisa menutup banyak tamu: wajib `allocations` (total = `amount`); 1 tamu → alokasi otomatis.
  - `rooms.current_occupants` + `status` disinkronkan dalam transaksi (`adjustRoomOccupancy`); check-in tolak (409) bila kapasitas penuh/maintenance.
  - Pindah kamar: update `room_id` in-place + history `pindah` (note `"lama -> baru"`); tidak ada baris tamu baru.
  - Checkout: isi `checkout_date`/`checkout_time` + history `check-out`; ganda → 409.
  - `DELETE` payment = void → `status='batal'` (409 bila sudah batal); `PUT` payment `amount` < total alokasi → 409.
  - `DELETE` room dengan tamu aktif → 409; `DELETE` payment-method/purchase-source yang dipakai → 409.
  - Unique kamar: index **komposit** `uq_rooms_loc_number` `(location_id, room_number)` — tag `uniqueIndex:uq_rooms_loc_number` di **kedua** field `LocationID`+`RoomNumber`. JANGAN unique single-kolom `room_number` (produksi punya nomor sama di lokasi berbeda → AutoMigrate gagal `Error 1062 Duplicate entry 'L1'`).
  - Broadcast WebSocket (`handler/ws.go`): setelah commit, event `type=room` (action create/update/status/delete + occupancy) dan `type=guest` (checkin/checkout/pindah) dikirim ke semua client `/ws` yang terautentikasi.
  - Struk: `GET /penginapan/payments/:id/receipt` **public** (HTML 78mm, handler `PaymentReceipt` di `penginapan_receipt.go`); nama hotel dari config `nama_hotel` (fallback `PENGINAPAN`), baris per tamu dari alokasi payment, status `batal` → cap `DIBATALKAN`, `?autoprint=1` → `window.print()`. Error (400/404/500) tetap format standar.
  - QRIS: `GET /penginapan/qris` (JWT) membaca config `qris_payload` (wajib; string EMV QR statis/URL) + opsional `qris_merchant`; belum diisi → 404 `NOT_CONFIGURED` dengan detail cara konfigurasi. Alur: check-in `payment.status=pending` → tampilkan QR → `PUT /payments/:id {"status":"lunas"}` → struk.

## Pekerjaan belum selesai (lanjutkan bila diminta)

- Test runtime sudah dilakukan via `scripts/testing/` (248 assert: 142 API + 5 env-flag + 87 penginapan + 14 WS, Docker lokal) terhadap semua endpoint yang ada; sisa yang belum teruji: jalur sukses fallback API tiket & Damri (butuh akses jaringan keluar).
