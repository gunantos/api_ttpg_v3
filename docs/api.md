# Dokumentasi API — TTPG

Base URL (default): `http://localhost:3080`

> Server listen di `0.0.0.0` — dari komputer lain di LAN yang sama, ganti `localhost` dengan IP server, mis. `http://192.168.1.50:3080` (lihat README → "Akses dari Komputer Lain" untuk buka firewall).

Sumber kebenaran route: `internal/routes/routes.go`. **Update file ini setiap ada perubahan endpoint.**

> **Halaman docs interaktif**: server menyajikan dokumentasi HTML (parameter, body, contoh respons, snippet cURL/PHP/Python/Node.js/Go, tombol "Coba") di **`GET /docs`** — source: `internal/docs/api.html` (di-embed via `internal/docs/docs.go`). File markdown ini tetap sumber ringkas.

---

## Daftar Endpoint

| Method | Path | Auth |
|--------|------|------|
| GET | `/boarding/print/:code` | Tanpa auth (public) |
| GET | `/boardingpass/print/:code` | Tanpa auth (public, alias lama dari `/boarding/print/:code`) |
| GET | `/penginapan/payments/:id/receipt` | Tanpa auth (public) — HTML struk pembayaran 78mm, `?autoprint=1` |
| GET | `/ws` | JWT via query `?token=` (public route, auth di handler) |
| GET | `/docs` | Tanpa auth (public) — halaman dokumentasi interaktif (HTML, contoh kode cURL/PHP/Python/Node/Go) |
| GET | `/activity` | Tanpa auth (public) — halaman riwayat log request user (HTML; data diambil via `?token=`) |
| GET | `/activity/logs` | Seragam† — API log (JSON, paginasi + filter) |
| GET | `/vending/config?machine_id=` | Seragam† — bundle config mesin (auto-register) |
| POST | `/vending/device/status` | Seragam† — heartbeat mesin (30 detik) |
| GET | `/vending/media` | Seragam† — katalog media attract (opsional) |
| POST | `/gate` | Seragam† |
| POST | `/checkin-mandiri` | Seragam† |
| GET | `/boardingpass` | Seragam† |
| GET | `/boardingpass/:id` | Seragam† |
| POST | `/boardingpass` | Seragam† |
| PUT | `/boardingpass/:id` | Seragam† |
| DELETE | `/boardingpass/:id` | Seragam† |
| POST | `/auth` | Tanpa auth |
| GET | `/tujuan` | Seragam† |
| GET | `/po` | Seragam† |
| GET | `/boarding` | Seragam† |
| POST | `/boarding` | Seragam† |
| PUT | `/boarding/:id` | Seragam† |
| DELETE | `/boarding/:id` | Seragam† |
| GET/POST | `/penginapan/configs`, `/penginapan/configs/key/:key` | JWT (AuthMiddleware) |
| PUT/DELETE | `/penginapan/configs/:id` | JWT (AuthMiddleware) |
| GET/POST/PUT/DELETE | `/penginapan/payment-methods[/:id]` | JWT (AuthMiddleware) |
| GET/POST/PUT/DELETE | `/penginapan/purchase-sources[/:id]` | JWT (AuthMiddleware) |
| GET/POST/PUT/DELETE | `/penginapan/locations[/:id]` | JWT (AuthMiddleware) |
| GET/POST/PUT/DELETE | `/penginapan/rooms[/:id]`, `PUT /penginapan/rooms/:id/status` | JWT (AuthMiddleware) |
| GET/POST/PUT/DELETE | `/penginapan/room-layouts[/:id]` | JWT (AuthMiddleware) |
| GET/POST/PUT | `/penginapan/guests`, `/penginapan/guests/:id` (GET/PUT/POST via sub-path) | JWT (AuthMiddleware) |
| POST | `/penginapan/guests/:id/checkout`, `/penginapan/guests/:id/pindah` | JWT (AuthMiddleware) |
| GET | `/penginapan/guests/:id/histories` | JWT (AuthMiddleware) |
| POST/GET/DELETE | `/penginapan/guests/:id/ktp` | JWT (AuthMiddleware) — foto KTP (multipart file ≤5MB; GET = bytes gambar; hapus) |
| GET/POST/PUT/DELETE | `/penginapan/payments[/:id]` | JWT (AuthMiddleware) |
| POST | `/penginapan/payments/:id/pay` | JWT — buat tagihan payment gateway (Midtrans/Xendit) |
| GET | `/penginapan/payment-gateway` | JWT — info gateway aktif (provider, configured) |
| GET | `/penginapan/qris` | JWT (AuthMiddleware) |
| GET | `/penginapan/room-histories` | JWT (AuthMiddleware) |

Total 46 endpoint penginapan: 43 di group `/penginapan` + 3 public (`GET /penginapan/payments/:id/receipt`, `POST /penginapan/webhooks/midtrans`, `POST /penginapan/webhooks/xendit`) — detail dan body request di bagian [Penginapan](#penginapan).

---

## Autentikasi

† **Seragam** — semua route protected (gate, checkin-mandiri, boardingpass, tujuan, po, boarding, penginapan) memakai satu middleware `auth.AuthMiddleware` yang menerima salah satu skema:

| Skema | Header | Catatan |
|-------|--------|---------|
| API key (DB `ttpg_users`) | `X-ttpg-key: dev-key-123` atau `X-API-Key: …` | cocok `user.secret_key` (user aktif; user_id/role/po_ids dari baris `user` tersebut) |
| API key literal | `X-ttpg-key: 12345` atau `X-API-Key: 12345` | cocok literal dengan env `API_KEY` (default `12345`) — berlaku sebagai superkey; konteks `user_id=0`, `role=api_key` |
| JWT | `Authorization: Bearer <token>` | token dari `POST /auth`, berlaku 1 minggu |
| Basic | `Authorization: Basic base64(username:password)` | user aktif di `ttpg_users.user` (bcrypt PHP `$2y$` kompatibel) |

Digest **dilepas** (tabel `user` di `ttpg_users` tidak punya kolom HA1); skema tak dikenal → `401`.

Urutan validasi: header API key (literal env `API_KEY` dulu, baru `user.secret_key`) → `Authorization` (`Bearer` / `Basic`). Gagal atau skema tidak dikenal → `401` + header `WWW-Authenticate: Basic realm="ttpg"`:

```json
HTTP 401
{"success": false, "message": "…"}
```

> Catatan: `X-ttpg-key` tetap punya dua makna dalam satu middleware — literal env `API_KEY` (superkey, `user_id=0`) dicek lebih dulu, baru `user.secret_key` di `ttpg_users`.

### Aturan endpoint (DB `ttpg_users`)

Setelah identitas valid, middleware mengecek baris `api_url` yang cocok **method + path** (kolom `controller` ternormalisasi: huruf kecil, tanpa trailing `/`, `/x/index` == `/x`; dibandingkan juga dengan pola route Gin, mis. `/penginapan/rooms/:id`):

- `authorization` = skema wajib (`api_key` / `bearer_token` / `basic_auth` / `none`) → salah skema → **403**;
- izin akses (dilewati bila `user.all_access='y'` atau superkey env `API_KEY`): gabungan `user.allowed` + `access.id_api` (dipisah `;` **atau** `,`) wajib memuat id baris `api_url`, dan `access.block='n'` + `access.allow='y'` → salah → **403**;
- `user.is_block=1` / `is_approved=0` → **403** (login/key ditolak **401**).

Rute **tanpa** baris `api_url` menerima skema apa pun (aturan hanya berlaku bila baris ada — keputusan: ketat saat ada, bebas saat tidak ada).

Setiap request (termasuk yang gagal) dicatat **async** ke `ttpg_users.logs`: `uri`, `method`, `params` (JSON `{"query","body"}` — query & body sudah **di-redaksi**: field rahasia seperti `password`/`secret`/`token`/`api_key` jadi `[REDACTED]`; body dibaca maks 16KB), `api_key`, `ip_address`, `time`, `rtime`, `authorized` (`y`/`n`), `response_code`, `id_user`, `po_id`, `access_by` (skema), `date`. Log **dihapus otomatis** lebih tua dari `LOG_RETENTION_DAYS` (default 30 hari; `0` = nonaktif) — pembersihan jalan tiap jam dan saat server start. Lihat bagian [Aktivitas & Log](#aktivitas--log-tampilan-user).

Endpoint **public** (tanpa auth): `POST /auth`, `/boarding/print/:code` (+alias), `/penginapan/payments/:id/receipt`, `/penginapan/webhooks/midtrans`, `/penginapan/webhooks/xendit`, `/ws` (JWT via query `?token=`), `/docs`, `/activity`.

---

## Format Response

Dua format, tergantung endpoint:

**A. Format standar** (`internal/pkg/response`) — untuk `POST /auth`, `/tujuan`, `/po`, `/boarding*`, **seluruh `/penginapan/*`**:

```json
{
  "success": true,
  "message": "…",
  "data": …,
  "pagination": { "page": 1, "limit": 10, "total": 0, "total_pages": 0 }   // opsional
}
```

Error:

```json
{
  "success": false,
  "message": "…",
  "errors": [{ "field": "…", "code": "…", "detail": "…" }]               // opsional
}
```

**B. Format mentah** — untuk `/gate`, `/checkin-mandiri`, `/boardingpass*` (lihat masing-masing endpoint).

---

## Auth

### POST `/auth`

Request:

```json
{ "username": "admin", "password": "admin123" }
```

Response `200`:

```json
{
  "success": true,
  "message": "Login berhasil",
  "data": {
    "token": "<jwt>",
    "user_id": 1,
    "role": "admin",
    "expire": "2026-10-14T10:00:00+07:00"
  }
}
```

`401` — username/password salah (format error standar).

> **Perubahan route**: login sebelumnya `POST /auth/login` — sekarang hanya **`POST /auth`** (path lama → `404`). `GET /auth/key` **dihapus**; API key = kolom `user.secret_key` (lihat [Autentikasi](#autentikasi)).

---

## Gate

### POST `/gate`

Auth seragam†. Body:

```json
{ "noreg": "ABC123", "gate_id": 1 }
```

Response `200` selalu memakai shape:

```json
{ "success": true, "reason": "VALID CHECKIN" }
```

| `reason` | Keterangan |
|----------|------------|
| `BYPASS MODE` | env `BYPASS=true`, selalu lolos |
| `VALID via LIST_ID_CARD` | noreg ada di env `LIST_ID_CARD` |
| `VALID via API TICKET` | noreg tidak ada lokal → diambil dari API tiket eksternal, record baru dibuat (`PO122`, `fungsi=gatein`) |
| `VALID CHECKIN` | `fungsi` = `checking`/`checkin`/kosong → lolos, di-update jadi `gatein` |
| `VALID RESCAN` | `fungsi` = `gatein`/`gateout`, scan ≤ 5× dan belum expired 5 menit |
| `MAXIMUM SCAN TERCAPAI` | `count_scan >= 5` → tolak |
| `SCAN SUDAH EXPIRED` | scan terakhir > 5 menit lalu → tolak |
| `STATUS BOARDINGPASS TIDAK VALID` | `fungsi` lain → tolak |
| `TIMEOUT SISTEM TIKET` / `KONEKSI KE SISTEM TIKET GAGAL` / `BOARDINGPASS TIDAK DITEMUKAN` | fallback API tiket gagal (`success: false`) |

Error:

```json
HTTP 400  { "error": "noreg wajib diisi" }
HTTP 500  { "error": "…" }
```

Header opsional: `X-Device-ID` (default: IP client).  
Query: tidak ada — `gate_id` dari body.

---

## Checkin Mandiri

### POST `/checkin-mandiri`

Auth seragam†. Body (JSON atau form):

```json
{ "ticket_id": "TKT-001" }
```

Respons:

| Status | Body |
|--------|------|
| `201` | `{"status": true, "id": <mysql_id>, "noreg": "CM…", "ticket_id": "…", "source": "mongodb"\|"damri", "pembelian_dari": "…", "print_url": "…", "message": "checkin mandiri berhasil"}` |
| `200` | boardingpass ticket_id hari ini sudah `fungsi=checkin` → `{"status": true, "message": "boardingpass sudah checkin hari ini", "data": {…terminalBoardingpass…, "print_url"}}` |
| `409` | sudah ada hari ini tapi bukan checkin → `{"status": false, "message": "boardingpass dengan ticket_id ini sudah ada hari ini", "data": {…}}` |
| `404` | `{"status": false, "message": "ticket_id tidak ditemukan"}` (damri aktif: `…tidak ditemukan di database maupun damri`) |
| `400` | body tidak valid / `ticket_id wajib diisi` |
| `504` | `{"status": false, "message": "request damri timeout"}` |
| `500` | `{"status": false, "message": "…"}` |

Alur: cari di Mongo `boarding` (`status=active`, `user_id` tidak di `IgnoredMongoUserIDs`) → jika kosong dan `ENABLE_DAMRI_FALLBACK=true` → API Damri (`PO028`) → insert MySQL `terminal_boardingpass` (`fungsi=checkin`, `count_scan=0`) + insert koleksi Mongo `boardingpass` → commit; bila gagal di tengah, rollback MySQL & Mongo.

`data` (untuk 200/409):

```json
{
  "id": 1, "noreg": "…", "ticket_id": "…", "id_po": "…", "id_tujuan": 1,
  "nopol": "B 1234 …", "tanggal_berangkat": "2026-10-12",
  "jam_berangkat": "08:00:00", "nama_pnp": "…", "sex_pnp": "L",
  "fungsi": "gatein", "id_user": "…", "pembelian_dari": "…",
  "nama_po": "…", "tujuan": "…", "print_url": "http://…/boarding/print/CM…?…"
}
```

---

## Boardingpass

### GET `/boardingpass` (list)

Auth seragam†. Query (opsional): `noreg` (LIKE), `id_po`, `fungsi`, `id_tujuan` (int), `limit` (1–1000, default 100), `offset` (default 0). Hanya baris `deleted IS NULL`, urut `id DESC`.

`200`:

```json
{
  "data": [
    {"id": 6, "noreg": "…", "id_po": "PO001", "id_tujuan": 2, "nopol": "…",
     "tanggal_berangkat": "2026-10-13T00:00:00Z", "jam_berangkat": "08:30:00",
     "nama_pnp": "…", "fungsi": "checkin", "count_scan": 0,
     "created": "2026-10-13 04:35:59"}
  ],
  "total": 1
}
```

`total` = jumlah baris yang cocok (tanpa limit/offset). `400` bila `limit`/`offset`/`id_tujuan` tidak valid; `401` tanpa API key.

### GET `/boardingpass/:id`

Auth seragam†. Response `200` = objek `model.Boardingpass` apa adanya:

```json
{
  "id": 1, "noreg": "…", "ticket_id": "…", "id_po": "…", "id_tujuan": 1,
  "nopol": "…", "tanggal_berangkat": "2026-10-12", "jam_berangkat": "08:00:00",
  "nama_pnp": "…", "sex_pnp": {"String": "L", "Valid": true},
  "fungsi": {"String": "checkin", "Valid": true},
  "gate_id": {"Int64": 0, "Valid": false}, "device_id": {"String": "", "Valid": false},
  "time_tab": {"Time": "…", "Valid": false}, "id_user": {"String": "", "Valid": false},
  "retribusi": 0, "count_scan": {"Int64": 0, "Valid": true},
  "pembelian_dari": {"String": "…", "Valid": true},
  "created": "…", "updated": {"Time": "…", "Valid": false},
  "deleted": {"Time": "…", "Valid": false}
}
```

(`404` `{"error": "boardingpass tidak ditemukan"}`, `400` id tidak valid, `500` error query.)

### POST `/boardingpass`

Auth seragam†. Body:

```json
{
  "noreg": "ABC123",
  "ticket_id": "TKT-001",
  "id_po": "PO001",
  "id_tujuan": 1,
  "nopol": "B 1234 …",
  "tanggal_berangkat": "2026-10-12",
  "jam_berangkat": "08:00:00",
  "nama_pnp": "Budi",
  "sex_pnp": "L",
  "fungsi": "checking",
  "retribusi": 0,
  "pembelian_dari": "loket"
}
```

Wajib: `noreg`. Field kosong (`nama_pnp`, `tanggal_berangkat`, `jam_berangkat`, `nopol`) diambil dari API tiket eksternal.

Respons:

- `201` `{"id": 123, "message": "boardingpass berhasil dibuat", "api_data": {…data tiket…}}`
- `404` `{"error": "noreg tidak ditemukan di sistem tiket"}`
- `409` `{"error": "boardingpass dengan noreg ini sudah ada"}`
- `400`/`500` `{"error": "…"}`

#### Upload langsung saat create (opsional)

`POST /boardingpass` juga menerima `multipart/form-data` dengan field data yang sama (form field, bukan JSON) plus file opsional:

- `ktp` / `ktp_file` / `ktp_photo` → foto KTP
- `tiket` / `ticket` / `file` / `ticket_file` / `ticket_photo` → foto tiket

Jika file dikirim, upload dilakukan setelah baris boardingpass dibuat; respons `201` menambahkan `ktp_photo`/`ticket_photo` (nama file) bila sukses. Bila upload gagal, respons `4xx/5xx` menyertakan `id` boardingpass yang sudah dibuat agar file bisa diulang via endpoint terpisah.


### PUT `/boardingpass/:id`

Auth seragam†. Body — semua field opsional, hanya yang dikirim yang di-update:

```json
{"fungsi": "gateout", "gate_id": "1", "device_id": "DEV1", "time_tab": "2026-01-02 10:20:30", "count_scan": 3}
```

- `gate_id` string numerik; string kosong → `NULL`; non-numerik → `400`.
- `time_tab` format `YYYY-MM-DD HH:MM:SS` (juga menerima `YYYY-MM-DDTHH:MM:SS`, RFC3339, `YYYY-MM-DD HH:MM`); string kosong → `NULL`; salah format → `400`.
- `count_scan` harus ≥ 0.

Respons: `200` = objek boardingpass terbaru (format sama dengan `GET /boardingpass/:id`); `404` tidak ada/sudah dihapus; `400` body kosong (`"tidak ada field yang diupdate"`), id/gate_id/time_tab/count_scan tidak valid.

### DELETE `/boardingpass/:id`

Auth seragam†. Soft delete (`deleted = NOW()`, baris tetap ada tapi disembunyikan dari GET/list).

- `200` `{"message": "boardingpass berhasil dihapus"}`
- `404` `{"error": "boardingpass tidak ditemukan"}` (tidak ada atau sudah terhapus)
- `400` id tidak valid

### Gambar opsional boardingpass (KTP / tiket)

Auth seragam†. `:kind` pada path = `ktp` atau `tiket` (alias path statis `/boardingpass/:id/ktp` dan `/boardingpass/:id/tiket`). File disimpan di `$UPLOAD_DIR/boardingpass/{ktp|tiket}` dengan nama yang dibuat server: `bp<id>_<kind>_<ts>_<rand>.<ext>` (jpg/png/webp). Upload ulang mengganti file lama; `GET /boardingpass/:id` menampilkan `ktp_photo`/`ticket_photo` (null bila belum ada).

- Maks 5MB, format JPG/PNG/WebP (deteksi via content-type, bukan ekstensi).
- `GET` mengembalikan bytes gambar + `Content-Type` sesuai file (cache private 24 jam).
- Hanya boardingpass yang belum dihapus (`deleted IS NULL`).

#### POST `/boardingpass/:id/:kind` (upload)

Multipart `form-data`, field `file`.

- `200` `{"id": 123, "kind": "ktp", "ktp_photo": "bp123_ktp_…png", "message": "gambar berhasil disimpan"}`
- `400` id tidak valid, kind tidak valid, file wajib, ukuran >5MB, format bukan JPG/PNG/WebP
- `404` boardingpass tidak ditemukan / sudah dihapus
- `500` gagal simpan file/DB

#### GET `/boardingpass/:id/:kind` (ambil gambar)

- `200` bytes gambar
- `400` id/kind tidak valid
- `404` boardingpass tidak ada, gambar belum ada, atau file hilang di disk

#### DELETE `/boardingpass/:id/:kind` (hapus)

- `200` `{"id": 123, "kind": "ktp", "ktp_photo": null, "message": "gambar berhasil dihapus"}`
- `400` id/kind tidak valid
- `404` boardingpass tidak ditemukan / sudah dihapus

### GET `/boarding/print/:code`

**Public (tanpa auth).** Response `200` = HTML siap cetak (ukuran 78mm, QR dari `qrcodejs`). URL lama `GET /boardingpass/print/:code` tetap berfungsi sebagai alias.

`:code` = `noreg` **atau** `ticket_id` (lookup `terminal_boardingpass` WHERE deleted IS NULL).

Query param opsional — jika **semua** ini terisi, DB tidak dipanggil (untuk print instan dari checkin):

| Param | Keterangan |
|-------|------------|
| `nama_pnp` | nama penumpang |
| `nama_po` | nama PO |
| `tujuan` | tujuan |
| `tanggal_berangkat` | `YYYY-MM-DD` |
| `jam_berangkat` | `HH:MM:SS` |
| `ticket_id` | opsional |
| `autoprint` | `1` → auto `window.print()` saat load |

Error: `400` kode kosong, `404` boardingpass tidak ditemukan, `500` `{"status": false, "message": "…"}`.

---

## Master Data

### GET `/tujuan`

Auth seragam (JWT / Basic / API key — lihat [Autentikasi](#autentikasi)). Format standar, `data` = array `tbl_tujuan` semua kolom, urut `id_tujuan`.

```json
{ "success": true, "message": "Data tujuan ditemukan", "data": [ … ] }
```

### GET `/po`

Auth seragam (JWT / Basic / API key — lihat [Autentikasi](#autentikasi)). Hanya `id_po` dan `nama_po`, urut `id_po`.

```json
{ "success": true, "message": "Data po ditemukan",
  "data": [ { "id_po": "PO001", "nama_po": "…" } ] }
```

---

### GET `/bus`

Auth seragam (JWT / Basic / API key — lihat [Autentikasi](#autentikasi)). Data `tbl_bus` (MySQL `ttpg`) untuk autocomplete bus di frontend, dengan **JOIN** `terminal_trayek_main` (trayek) dan `tbl_tujuan` (asal & tujuan) agar rute/trayek dan kota asal-tujuan ikut terkirim. Query opsional:

- `q` / `search` — LIKE pada `kode`, `nopol`, `trayek1`, `trayek2`, `nm_perusahaan`, `id_po`
- `id_po` — filter exact PO
- `status` — filter exact (`1` = aktif)
- `limit` — 1–100, default 20

Urut `nopol ASC, id_bus ASC`.

`200`:

```json
{ "success": true, "message": "Data bus ditemukan",
  "data": [ { "id_bus": 4, "kode": "101010", "nopol": "B7013BGA", "id_po": "PO009",
              "nm_perusahaan": "PT. ARIMBI JAYA AGUNG", "trayek1": "…", "trayek2": "…",
              "jenis_bus": "ac", "pnp": 44, "status": "1",
              "trayek_main_id": 3, "trayek_kode": "0000000000",
              "asal_id": 18, "asal": "Jakarta (Pulo Gebang)",
              "tujuan_id": 643, "tujuan": "…",
              "rute": "MERAK-TOL MERAK-…-JAKARTA(PULO)" } ] }
```

`400` bila `limit` tidak valid.

---

## Boarding (MongoDB)

Auth seragam (JWT / Basic / API key — lihat [Autentikasi](#autentikasi)). Koleksi `boarding` (db `ttpg`).

Aturan `po_id` user: `ttpg_users.user.po_id` disimpan sebagai string dipisah koma (atau `;`) → di-parse `[]string`; nilai dibaca segar dari DB per request (JWT cukup sebagai bukti login).

- 0 po_id → bebas semua PO
- 1 po_id → auto-fill `po_id` saat POST, filter otomatis saat GET
- >1 po_id → `po_id` wajib sesuai daftar; selain itu `403`

### GET `/boarding`

Query: `po_id`, `ticket_id`, `destination_id` (opsional, disaring ke `po_id` user), `limit` (1-1000, default 100), `offset` (default 0).

`destination_id` menerima nilai numerik; dokumen lama dengan nilai string (`"395"`) atau object (`{id_tujuan: 395, …}`) ikut cocok. `destination_id` non-numerik → `400`. `limit` di luar 1-1000 atau `offset` negatif → `400`.

Respons (paginasi menyertai `data`):

```json
{
  "success": true,
  "message": "Data boarding ditemukan",
  "data": [ … ],
  "pagination": { "page": 1, "limit": 100, "total": 1235253, "total_pages": 12353 }
}
```

### POST `/boarding`

```json
{
  "po_id": "PO001",
  "ticket_id": "TKT-001",
  "nopol": "B 1234 …",
  "destination_id": 1,
  "date_of_departure": "2026-10-12",
  "time_of_departure": "08:00:00",
  "nik": "…", "name": "Budi", "gender": "L",
  "address": "…", "city": "…", "province": "…",
  "telp": "…", "email": "…"
}
```

Wajib: `ticket_id`, `destination_id`, `date_of_departure`, `time_of_departure`, `name`.  
`name` boleh multi (pisah `,` atau `;`) → jadi array `pnp`; `gender` dipasangkan per index.

Respons `201`:

```json
{ "success": true, "message": "Boarding berhasil dibuat", "data": { …doc, "_id": "…" } }
```

Error: `400` validasi, `403` po_id tidak diizinkan, `500` DB.

### PUT `/boarding/:id`

`:id` = ObjectId hex. Body (semua opsional): `nopol`, `status`, `date_of_departure`, `time_of_departure`, `destination_id`.

- `200` `{"success": true, "message": "Boarding berhasil diupdate", "data": null}`
- `400` id tidak valid, `404` tidak ditemukan, `403` po_id tidak diizinkan

### DELETE `/boarding/:id`

- `200` `{"success": true, "message": "Boarding berhasil dihapus", "data": null}`
- `400`/`403`/`404` seperti di atas

---

## Penginapan

Auth seragam (`auth.AuthMiddleware`: JWT / Basic / API key) untuk route `/penginapan/*` — **kecuali** `GET /penginapan/payments/:id/receipt` dan webhook gateway `POST /penginapan/webhooks/{midtrans,xendit}` (public, dipanggil gateway tanpa header auth; struk dibuka via `window.open`). Format response: **standar A** (`success/message/data`). Database khusus `ttpg_penginapan` (env `PENGINAPAN_DSN`), GORM dengan AutoMigrate; skema canonical: `migrations/mysql/002_create_ttpg_penginapan.sql` (baru) atau upgrade: `003_upgrade_penginapan_v2.sql` + `005_upgrade_payments_gateway.sql` (tanpa DROP data).

Aturan bisnis:

- **Lunas**: total alokasi dari payment berstatus `lunas` ≥ `harga` tamu. Payment `pending` **tidak dihitung**.
- **Satu payment → banyak tamu**: wajib `allocations` (`guest_index` saat check-in / `guest_id` saat manual); total alokasi = `amount` payment. 1 tamu → alokasi otomatis.
- **Occupancy** `rooms.current_occupants` + `status` (`kosong`/`terisi`/`maintenance`) disinkronkan dalam transaksi. Ketersediaan client baca field turunan `available` di response kamar (lihat [Locations, rooms](#locations-rooms-room-layouts)).
- **Pindah kamar** mengubah `room_id` tamu in-place + history `pindah` (note `lama -> baru`); checkout mengisi `checkout_date`/`checkout_time` + history `check-out`.

### Configs & lookups

| Method & path | Keterangan |
|---------------|------------|
| `GET /penginapan/configs`, `GET /configs/key/:key` · `POST /configs` · `PUT/DELETE /configs/:id` | Config key/value (`key_configs`, `value_configs`) |
| `GET/POST/PUT/DELETE /penginapan/payment-methods[/:id]` | Seed: `cash`, `qris`, `transfer`, `debit`, `lainnya`; DELETE dipakai payment → 409 |
| `GET/POST/PUT/DELETE /penginapan/purchase-sources[/:id]` | Seed: `langsung`, `mesin`; DELETE dipakai payment → 409 |

### Locations, rooms, room-layouts

- `GET/POST/PUT/DELETE /penginapan/locations[/:id]` — `name`, `keterangan`.
- `GET/POST/PUT/DELETE /penginapan/rooms[/:id]` — `location_id`, `room_number` (unik per lokasi), `room_type` (`pria`/`wanita`), `capacity`. `GET /rooms` filter: `status`, `location_id`, `room_type`.
- **Field turunan `available`** di **semua** objek kamar (list/detail/CRUD/status, `room` embedded di tamu/layout, event WS): `true` bila `status=kosong` dan `current_occupants < capacity` — client cukup membaca field ini untuk ketersediaan (implementasi: custom `MarshalJSON` di `PenginapanRoom`, otomatis ikut serialisasi). Contoh respons `GET /penginapan/rooms`:

```json
{
  "success": true,
  "message": "Data kamar ditemukan",
  "data": [
    {
      "id": 1, "location_id": 1,
      "room_number": "201", "room_type": "pria",
      "capacity": 2, "current_occupants": 1,
      "status": "terisi", "available": false,
      "location": {"id": 1, "name": "Gedung A"}
    }
  ]
}
```
- **Aturan UI pilih kamar (check-in, aturan frontend)**: semua kamar selalu ditampilkan; kamar dengan `room_type` ≠ gender tamu → **disabled** (bukan warna hijau, tidak bisa diklik); hijau + bisa dipilih hanya `room_type` sama dengan gender tamu **dan** `available:true`.
- `PUT /penginapan/rooms/:id/status` — `{"status": "kosong"|"terisi"|"maintenance"}`.
- `DELETE /rooms` ditolak (409) bila ada tamu aktif.
- `GET/POST/PUT/DELETE /penginapan/room-layouts[/:id]` — `location_id`, `room_id`, `row_number`, `urutan`; `GET` filter `location_id` (preload `location`, `room`).

### Guests (check-in / check-out / pindah)

`GET /penginapan/guests` — filter `status=inhouse|out`, `room_id`, `checkin_date` (limit 500). Setiap tamu disertai summary pembayaran:

```json
{ "id": 1, "room_id": 1, "name": "Budi", "nik": "3175...", "ktp_photo": null, "harga": 100000,
  "room": { "room_number": "201", "available": false },
  "total_bayar": 100000, "sisa": 0, "lunas": true }
```

`GET /penginapan/guests/:id` — detail + `room` + summary; `PUT /penginapan/guests/:id` — update `name`/`nik`/`gender`/`harga` (harga baru dihitung ulang terhadap pembayaran).

`POST /penginapan/guests` (check-in, transaksional):

```json
{
  "guests": [
    { "room_id": 1, "name": "Budi", "nik": "…", "gender": "pria", "harga": 100000,
      "checkin_date": "2026-10-13", "checkin_time": "14:00" }
  ],
  "payment": {
    "amount": 180000, "payment_method_id": 1, "source_id": 1, "status": "pending",
    "allocations": [ { "guest_index": 0, "amount": 90000 }, { "guest_index": 1, "amount": 90000 } ]
  }
}
```

`payment` opsional; `status` default `lunas`. Respons `201`:

```json
{
  "success": true,
  "message": "Check-in berhasil",
  "data": {
    "guests": [
      { "id": 1, "room_id": 1, "name": "Budi", "gender": "pria", "harga": 100000,
        "checkin_date": "2026-10-13T00:00:00Z", "checkin_time": "14:00:00",
        "checkout_date": null, "checkout_time": null,
        "total_bayar": 100000, "sisa": 0, "lunas": true }
    ],
    "payment": {
      "id": 1, "amount": 100000, "status": "lunas", "paid_at": "…",
      "source": { "id": 1, "code": "langsung" },
      "payment_method": { "id": 1, "code": "cash" },
      "guests": [ { "guest_id": 1, "amount": 100000 } ]
    }
  }
}
```

Error: `400` validasi/alokasi, `404` room tidak ada, `409` kapasitas penuh / maintenance.

`POST /penginapan/guests/:id/checkout` — body `{"checkout_date":"YYYY-MM-DD","checkout_time":"HH:MM"}` (opsional, default sekarang). `409` bila sudah check-out.

`POST /penginapan/guests/:id/pindah` — `{"room_id": 4}` → `{"guest_id", "room_lama", "room_baru"}`. `400` kamar sama, `409` penuh/maintenance.

`GET /penginapan/guests/:id/histories` — history `check-in`/`check-out`/`pindah` tamu.

**Foto KTP**: `POST /penginapan/guests/:id/ktp` — multipart form-field `file` (JPG/PNG/WebP, maks 5MB) → `{"guest_id","ktp_photo"}`; upload ulang mengganti file lama. `GET .../ktp` — bytes gambar (butuh JWT; tampilkan via fetch-blob, bukan `<img src>`); `404` bila belum ada. `DELETE .../ktp` — hapus foto. Nama file dibuat server, disimpan di `$UPLOAD_DIR/ktp` (default `./uploads/ktp`).

### Payments

`GET /penginapan/payments` — filter `guest_id`, `status`, `source_id`, `payment_method_id` (preload `guests`, `source`, `payment_method`). `GET /penginapan/payments/:id`.

`POST /penginapan/payments` (manual, di luar check-in):

```json
{
  "amount": 50000, "payment_method_id": 1, "source_id": 1, "status": "lunas",
  "guest_ids": [3], "allocations": [], "note": "DP"
}
```

`guest_ids` wajib ≥ 1; 1 tamu → alokasi otomatis penuh; multi tamu → `allocations` (`guest_id` + `amount`, total = `amount`).

`PUT /penginapan/payments/:id` — update `amount`/`payment_method_id`/`source_id`/`status`/`note`. `409` bila `amount` < total alokasi. `status` `lunas` mengisi `paid_at`.

`DELETE /penginapan/payments/:id` — void (status `batal`); alokasi tidak lagi dihitung lunas. `409` bila sudah `batal`.

### Struk & QRIS

`GET /penginapan/payments/:id/receipt` — **public** (tanpa auth), mengembalikan HTML struk pembayaran 78mm siap cetak. Isi: nama hotel (config `nama_hotel`, fallback `PENGINAPAN`), nomor & status payment, metode/sumber/catatan, baris per tamu dari alokasi (nama, kamar, jumlah) + `TOTAL`; status `batal` diberi cap `DIBATALKAN`. Query `?autoprint=1` → `window.print()` otomatis saat load (pola sama seperti `/boarding/print`). `404` bila payment tidak ada.

`GET /penginapan/qris` — payload QRIS untuk ditampilkan ke tamu. Diambil dari config key `qris_payload` (string EMV QR statis / URL gambar QRIS); config opsional `qris_merchant` disertakan bila ada. `404` `NOT_CONFIGURED` bila `qris_payload` belum diisi.

```json
{ "success": true, "message": "Konfigurasi QRIS ditemukan",
  "data": { "payload": "00020101021126610014COM.GO-JEK.WWW0…", "merchant": "Nama Merchant" } }
```

### Payment gateway (Midtrans & Xendit)

Pembayaran online **QRIS** via **Midtrans** atau **Xendit** — provider dipilih lewat config DB (`ttpg_penginapan.configs`), arsitektur pluggable (`internal/payment/`). Alurnya murni QR langsung (bukan halaman invoice Snap): charge mengembalikan `qr_string` → frontend render QR → tamu scan → webhook menandai lunas. Fokus saat ini **QRIS + cash**; VA dapat ditambahkan nanti (kode VA saja, tanpa halaman hosted) tanpa mengubah handler. Provider baru cukup tambah file gateway + `Register` tanpa mengubah handler.

**Konfigurasi** (via `POST /penginapan/configs`, key → value):

| Key | Keterangan |
|---|---|
| `payment_gateway` | `midtrans` atau `xendit` — provider aktif (kosong = gateway nonaktif) |
| `midtrans_server_key` | Server key Midtrans (wajib; charge QRIS + verifikasi webhook) |
| `midtrans_env` | `sandbox` (default) atau `production` |
| `midtrans_api_url` | Override base URL (opsional; untuk mock/test) |
| `xendit_secret_key` | Secret key Xendit (wajib; buat dynamic QR + verifikasi) |
| `xendit_callback_token` | Token `X-Callback-Token` webhook Xendit (wajib utk verifikasi) |
| `xendit_api_url` | Override base URL (opsional; untuk mock/test) |
| `payment_gateway_callback_url` | URL finish/callback frontend (opsional) |

**Alur**:

1. `GET /penginapan/payment-gateway` → `{provider, configured, webhook_path}` — frontend cek gateway aktif.
2. Check-in / `POST /penginapan/payments` dengan `status: "pending"`.
3. `POST /penginapan/payments/:id/pay` → charge QRIS ke gateway → `{external_id, qr_string, expires_at}` — frontend render `qr_string` sebagai gambar QR (mis. lib qrCode) lalu tamu scan. Idempoten: QRIS pending yang belum kedaluwarsa dikembalikan ulang tanpa charge baru. `404 NOT_CONFIGURED` bila gateway belum diset; `409` bila payment `lunas`/`batal`; `502 GATEWAY_ERROR` bila API gateway gagal.
4. Gateway memanggil webhook public:
   - **Midtrans** `POST /penginapan/webhooks/midtrans` — API Midtrans QRIS (`POST /v2/charge`, body `{payment_type:"qris", transaction_details:{order_id,gross_amount}}`); verifikasi `signature_key` = SHA512(`order_id` + `status_code` + `gross_amount` + server key) — **urutan wajib sesuai docs**; `settlement`/`capture` → payment **lunas** (+ `paid_at`); `cancel`/`deny` → `gateway_status` saja; `expire` → `gateway_status`. `gross_amount` wajib sama dengan `amount` payment (selisih → `403`).
   - **Xendit** `POST /penginapan/webhooks/xendit` — API Xendit QR Code (`POST /qr_codes`, dynamic; auth Basic + header `api-version: 2022-07-31`); verifikasi header `X-Callback-Token`; status `PAID`/`SETTLED` → **lunas**; `FAILED`/`EXPIRED` → `gateway_status` saja (payment tetap `pending`).
   - Signature/token salah → `401`; `external_id` tak dikenal → `404`; webhook ganda saat sudah lunas → `200` idempoten.
5. Setelah lunas (webhook) → buka `GET /penginapan/payments/:id/receipt?autoprint=1`.

Kolom gateway di tabel `payments` (AutoMigrate; upgrade DB lama: `migrations/mysql/005_upgrade_payments_gateway.sql`): `gateway`, `external_id` (index), `qr_string`, `gateway_status`, `gateway_expires_at` (legacy `invoice_url`/`snap_token` tidak dipakai alur QRIS) — semua ikut ter-serialize di response `GET/POST /penginapan/payments`.

**Hold pembayaran cash (timeout otomatis):** payment `pending` metode **cash** tanpa gateway yang lebih tua dari config `payment_hold_minutes` (menit; default **15**; `0` = nonaktif; set via `POST/PUT /penginapan/configs`) otomatis → `status='batal'` **dan** tamu terkait yang belum lunas di-check-out otomatis (history `check-out otomatis: pembayaran timeout`) sehingga kamar langsung `available` + event WS terkirim. Sweeper jalan tiap 60 detik. Payment yang sudah di-charge gateway tidak kena aturan ini (expiry ikut provider).

### Alur sekali jalan (front-desk)

Check-in + pembayaran + struk/QRIS dapat diselesaikan dengan sedikit request:

1. `GET /penginapan/rooms?status=kosong` — pilih kamar; `GET /penginapan/payment-methods` — pilih metode.
2. Satu request: `POST /penginapan/guests` dengan `guests[]` (nama, nik, gender, harga, room_id) + `payment` (`amount`, `payment_method_id`, `source_id`, opsional `status`/`allocations`). Respons memuat `data.payment.id` dan `data.payment.payment_method.code`.
3. Cabut sesuai metode:
   - **cash** (`code=cash`, default `status=lunas`) → buka `GET /penginapan/payments/{id}/receipt?autoprint=1` → selesai.
   - **qris** → `GET /penginapan/qris`, tampilkan QR ke tamu (render payload di frontend), check-in dengan `payment.status="pending"`; setelah lunas → `PUT /penginapan/payments/{id}` `{"status":"lunas"}` → buka struk.
   - **midtrans/xendit** (gateway aktif, lihat [Payment gateway](#payment-gateway-midtrans--xendit)) → check-in `payment.status="pending"` → `POST /penginapan/payments/{id}/pay` → render `qr_string` sebagai QR ke tamu → scan → tunggu webhook → buka struk.

### Room histories

`GET /penginapan/room-histories` — filter `room_id`, `guest_id`, `status`.

---

## Aktivitas & Log (tampilan user)

Halaman **`GET /activity`** (public, HTML) menampilkan riwayat request milik user: waktu, method, path, status, terautentikasi atau tidak, durasi, IP, skema akses, plus detail **query & body terkirim** (rahasia disamarkan `[REDACTED]`). Token JWT disimpan via `?token=<JWT>` (langsung dari URL) atau tempel manual di halaman (localStorage browser).

`GET /activity/logs` (JWT/BASIC/API key seragam) — API JSON di balik halaman:

| Query | Keterangan |
|---|---|
| `limit` / `offset` | `limit` 1–100 (default 20), `offset` default 0 — paginasi |
| `method` | `GET`/`POST`/`PUT`/`DELETE` |
| `code` | filter response_code persis (mis. `403`) |
| `path` | pencarian substring di `uri` |
| `authorized` | `y` / `n` |
| `from`, `to` | rentang tanggal `YYYY-MM-DD` (inclusive) |
| `user_id` | **admin saja** — filter log user tertentu; selain itu user biasa selalu hanya melihat log miliknya (`id_user`-nya sendiri) |

Respons memakai format standar + `pagination` (`page`, `limit`, `total`, `total_pages`); urutan terbaru dulu.

```json
{ "success": true, "message": "Log aktivitas ditemukan",
  "data": [ { "id": 64, "uri": "/penginapan/payments", "method": "POST",
      "params": "{\"query\":\"\",\"body\":\"{\\\"amount\\\":50000}\"}",
      "ip_address": "127.0.0.1", "rtime": 0.012, "authorized": "y",
      "response_code": 201, "id_user": 1, "access_by": "bearer_token",
      "date": "2026-10-15T10:30:00+07:00" } ],
  "pagination": { "page": 1, "limit": 20, "total": 64, "total_pages": 4 } }
```

`401` tanpa token / token kedaluwarsa.

---

## Vending (mesin OTOMAT)

Tiga endpoint untuk client mesin (auth seragam† — contoh memakai `X-TTPG-KEY`). Data di DB `ttpg`: `vending_machines`, `vending_configs`, `vending_media` — migration `migrations/mysql/006_create_ttpg_vending.sql` (**jalankan manual di produksi**, lokal otomatis via `reset.sh`).

**Auto-register**: `machine_id` yang belum ada di `vending_machines` dibuat otomatis dengan `status=pending` (approve mesin via admin panel — menyusul); heartbeat pun begitu.

**Config bundle** disimpan per baris: global `machine_id='__global__'` (di-seed migration: versi `cfg-20261015-1`, `updated_at` 2026-10-15 10:00:00) + opsional baris override per mesin. **Server yang meng-merge** (client tidak merge lokal): objek di-merge rekursif (field global yang tidak di-override tetap), array/scalar diganti oleh override; `version`/`updated_at` diambil dari baris override bila ada, selain itu dari global.

**Perilaku client**: setiap sync sukses (interval otomatis **5 menit**; manual `window.VENDING_CONFIG.sync()` atau panel admin) menyimpan `version`, mengunduh media (Bagian C), lalu mem-broadcast event `config_updated` ke renderer. **Validasi client** (gagal → fallback cache): root bertipe objek, `idle` objek, `attract` objek, dan `attract.sources` array.

### GET `/vending/config`

```http
GET /vending/config?machine_id=PENDING_1
X-TTPG-KEY: <key>
Content-Type: application/json
```

`machine_id` wajib (400 bila kosong). Response (dibungkus `data`):

```json
{
  "version": "cfg-20261015-1",
  "updated_at": "2026-10-15T10:00:00Z",
  "cameras": {"ktp": null, "person": null},
  "ocr": {"crop": null, "language": "eng+ind", "confidence_min": 0.7},
  "idle": {"minutes": 3, "enabled": true},
  "attract": {"mode": "hybrid", "sources": []},
  "speech": {"enabled": true, "model": "whisper-small", "language": "id"},
  "media_cache_limit_mb": 500
}
```

`cameras`/`ocr`/`idle`/`attract`/`speech`/`media_cache_limit_mb` = payload merge (bukan konstanta — bisa di-override per mesin). `sources` berisi URL media attract (diisi admin; juga tersedia via `GET /vending/media`). Setiap sync ikut memperbarui `last_seen_at` + `config_version` mesin.

### POST `/vending/device/status` (Heartbeat)

```http
POST /vending/device/status
X-TTPG-KEY: <key>
Content-Type: application/json
```

```json
{
  "machine_id": "PENDING_1",
  "app_version": "2.2.1-a",
  "config_version": "cfg-20261015-1",
  "last_sync_at": "2026-10-15T10:00:00.000Z",
  "extra": {"idle": 7, "person_present": true, "errors": []}
}
```

| Field | Wajib | Ket |
|---|---|---|
| `machine_id` | ya | auto-register bila belum ada |
| `app_version` | tidak | versi aplikasi mesin (maks 30 char) |
| `config_version` | tidak | versi config yang dipakai client (maks 50 char) |
| `last_sync_at` | tidak | RFC3339 — salah format → 400 |
| `extra` | tidak | objek JSON bebas (idle/person_present/errors, maks 8KB); non-objek → 400 |

Interval otomatis **30 detik**. Heartbeat mengisi `last_seen_at` (server time) dan tidak mengubah `status` mesin. Response: `data` `{machine_id, status}`.

### GET `/vending/media` (opsional)

Daftar konten attract — client boleh langsung memakai `attract.sources` dari bundle config; endpoint ini untuk pemisahan katalog media bila diperlukan. Hanya `enabled=1`, urut `order`.

```json
{ "success": true, "message": "Daftar media attract",
  "data": [ {"id": 1, "type": "video", "url": "https://.../a.mp4", "order": 1, "checksum": "sha256:..."} ] }
```

`401` tanpa key/token (ketiga endpoint).

---

## WebSocket

### GET `/ws?token=<JWT>`

Endpoint public untuk real-time update penginapan. Token JWT (hasil login `POST /auth`) dikirim sebagai query parameter; tanpa token atau token salah → `401` (respons HTTP sebelum upgrade). Client menerima push event setelah perubahan data di-commit; client tidak perlu mengirim pesan (ping/pong otomatis, ping tiap ±54 detik).

Format pesan (server → client):

```json
{"type": "room", "data": {"action": "status", "room": {"id": 1, "room_number": "201", "status": "maintenance", "current_occupants": 0, "available": false, "location": {...}}}}
```

| `type` | `action` | `data` |
|--------|----------|--------|
| `room` | `create` / `update` / `status` / `occupancy` | `room` — objek kamar lengkap (preload `location`), berisi `current_occupants` terbaru + `available` (turunan ketersediaan) |
| `room` | `delete` | `room_id` (int) |
| `guest` | `checkin` / `checkout` / `pindah` | `guest_ids` (array int) |

Event dikirim dari: `POST/PUT/DELETE /penginapan/rooms` (create/update/delete), `PUT /penginapan/rooms/:id/status` (status), check-in (occupancy per kamar + guest checkin), checkout, dan pindah kamar (guest pindah).

Contoh koneksi (test client resmi di `scripts/testing/wsclient`):

```bash
go run ./scripts/testing/wsclient "ws://127.0.0.1:3080/ws?token=$TOKEN" 15s
# CONNECTED
# EVENT type=room data={"action":"status","room":{...}}
```
