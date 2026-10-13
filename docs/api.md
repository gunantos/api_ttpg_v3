# Dokumentasi API — TTPG

Base URL (default): `http://localhost:8080`

> Server listen di `0.0.0.0` — dari komputer lain di LAN yang sama, ganti `localhost` dengan IP server, mis. `http://192.168.1.50:8080` (lihat README → "Akses dari Komputer Lain" untuk buka firewall).

Sumber kebenaran route: `internal/routes/routes.go`. **Update file ini setiap ada perubahan endpoint.**

---

## Daftar Endpoint

| Method | Path | Auth |
|--------|------|------|
| GET | `/boardingpass/print/:code` | Tanpa auth (public) |
| GET | `/ws` | JWT via query `?token=` (public route, auth di handler) |
| POST | `/gate` | API key literal |
| POST | `/checkin-mandiri` | API key literal |
| GET | `/boardingpass` | API key literal |
| GET | `/boardingpass/:id` | API key literal |
| POST | `/boardingpass` | API key literal |
| PUT | `/boardingpass/:id` | API key literal |
| DELETE | `/boardingpass/:id` | API key literal |
| POST | `/auth/login` | Tanpa auth |
| GET | `/auth/key` | Basic atau Digest |
| GET | `/tujuan` | JWT atau API key ter-hash |
| GET | `/po` | JWT atau API key ter-hash |
| GET | `/boarding` | JWT atau API key ter-hash |
| POST | `/boarding` | JWT atau API key ter-hash |
| PUT | `/boarding/:id` | JWT atau API key ter-hash |
| DELETE | `/boarding/:id` | JWT atau API key ter-hash |
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
| GET/POST/PUT/DELETE | `/penginapan/payments[/:id]` | JWT (AuthMiddleware) |
| GET | `/penginapan/room-histories` | JWT (AuthMiddleware) |

Total 41 endpoint penginapan di group `/penginapan` (detail dan body request di bagian [Penginapan](#penginapan)).

---

## Autentikasi

### 1. API key literal (group gate / checkin / boardingpass)

Header:

```
X-ttpg-key: 12345
# atau
X-API-Key: 12345
```

Nilai dibandingkan **literal** dengan env `API_KEY` (default `12345`). Jika tidak cocok:

```json
HTTP 401
{"status": false, "message": "api key tidak valid"}
```

### 2. JWT / API key ter-hash (group tujuan, po, boarding)

Opsi A — JWT (berlaku 1 minggu, didapat dari `/auth/login`):

```
Authorization: Bearer <token>
```

Opsi B — API key dari tabel `api_keys` (di-hash SHA-256):

```
X-ttpg-key: dev-key-123
```

Gagal:

```json
HTTP 401
{"success": false, "message": "…"}
```

> Catatan: header `X-ttpg-key` dipakai dua kali dengan makna berbeda — literal `API_KEY` di group gate/checkin, dan ter-hash ke `api_keys` di group tujuan/po/boarding.

### 3. Basic / Digest (`GET /auth/key`)

- Basic: `Authorization: Basic base64(username:password)`
- Digest: `Authorization: Digest …` (realm `ttpg`, `DigestHA1` user)

---

## Format Response

Dua format, tergantung endpoint:

**A. Format standar** (`internal/pkg/response`) — untuk `/auth/login`, `/tujuan`, `/po`, `/boarding*`, **seluruh `/penginapan/*`**:

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

### POST `/auth/login`

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

### GET `/auth/key`

Response `200`:

```json
{
  "status": true,
  "data": {
    "username": "admin",
    "po_id": "PO001",
    "production_key": "prod-key-123",
    "development_key": "dev-key-123"
  }
}
```

`401` — `{"status": false, "data": null, "message": "…"}` + header `WWW-Authenticate`.

---

## Gate

### POST `/gate`

Header: API key literal. Body:

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

Header: API key literal. Body (JSON atau form):

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
  "nama_po": "…", "tujuan": "…", "print_url": "http://…/boardingpass/print/CM…?…"
}
```

---

## Boardingpass

### GET `/boardingpass` (list)

Header: API key literal. Query (opsional): `noreg` (LIKE), `id_po`, `fungsi`, `id_tujuan` (int), `limit` (1–1000, default 100), `offset` (default 0). Hanya baris `deleted IS NULL`, urut `id DESC`.

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

Header: API key literal. Response `200` = objek `model.Boardingpass` apa adanya:

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

Header: API key literal. Body:

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

### PUT `/boardingpass/:id`

Header: API key literal. Body — semua field opsional, hanya yang dikirim yang di-update:

```json
{"fungsi": "gateout", "gate_id": "1", "device_id": "DEV1", "time_tab": "2026-01-02 10:20:30", "count_scan": 3}
```

- `gate_id` string numerik; string kosong → `NULL`; non-numerik → `400`.
- `time_tab` format `YYYY-MM-DD HH:MM:SS` (juga menerima `YYYY-MM-DDTHH:MM:SS`, RFC3339, `YYYY-MM-DD HH:MM`); string kosong → `NULL`; salah format → `400`.
- `count_scan` harus ≥ 0.

Respons: `200` = objek boardingpass terbaru (format sama dengan `GET /boardingpass/:id`); `404` tidak ada/sudah dihapus; `400` body kosong (`"tidak ada field yang diupdate"`), id/gate_id/time_tab/count_scan tidak valid.

### DELETE `/boardingpass/:id`

Header: API key literal. Soft delete (`deleted = NOW()`, baris tetap ada tapi disembunyikan dari GET/list).

- `200` `{"message": "boardingpass berhasil dihapus"}`
- `404` `{"error": "boardingpass tidak ditemukan"}` (tidak ada atau sudah terhapus)
- `400` id tidak valid

### GET `/boardingpass/print/:code`

**Public (tanpa auth).** Response `200` = HTML siap cetak (ukuran 78mm, QR dari `qrcodejs`).

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

Auth: JWT / API key ter-hash. Format standar, `data` = array `tbl_tujuan` semua kolom, urut `id_tujuan`.

```json
{ "success": true, "message": "Data tujuan ditemukan", "data": [ … ] }
```

### GET `/po`

Auth: JWT / API key ter-hash. Hanya `id_po` dan `nama_po`, urut `id_po`.

```json
{ "success": true, "message": "Data po ditemukan",
  "data": [ { "id_po": 1, "nama_po": "…" } ] }
```

---

## Boarding (MongoDB)

Auth: JWT / API key ter-hash. Koleksi `boarding` (db `ttpg`).

Aturan `po_id` user: `users.po_ids` disimpan sebagai string dipisah koma → di-parse `[]string`.

- 0 po_id → bebas semua PO
- 1 po_id → auto-fill `po_id` saat POST, filter otomatis saat GET
- >1 po_id → `po_id` wajib sesuai daftar; selain itu `403`

### GET `/boarding`

Query: `po_id`, `ticket_id`, `destination_id` (opsional, disaring ke `po_id` user).

```json
{ "success": true, "message": "Data boarding ditemukan", "data": [ … ] }
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

Auth: JWT (`AuthMiddleware`) untuk **semua** route `/penginapan/*`. Format response: **standar A** (`success/message/data`). Database khusus `ttpg_penginapan` (env `PENGINAPAN_DSN`), GORM dengan AutoMigrate; skema canonical: `migrations/mysql/002_create_ttpg_penginapan.sql` (baru) atau `migrations/mysql/003_upgrade_penginapan_v2.sql` (upgrade skema lama produksi, tanpa DROP data).

Aturan bisnis:

- **Lunas**: total alokasi dari payment berstatus `lunas` ≥ `harga` tamu. Payment `pending` **tidak dihitung**.
- **Satu payment → banyak tamu**: wajib `allocations` (`guest_index` saat check-in / `guest_id` saat manual); total alokasi = `amount` payment. 1 tamu → alokasi otomatis.
- **Occupancy** `rooms.current_occupants` + `status` (`kosong`/`terisi`/`maintenance`) disinkronkan dalam transaksi.
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
- `PUT /penginapan/rooms/:id/status` — `{"status": "kosong"|"terisi"|"maintenance"}`.
- `DELETE /rooms` ditolak (409) bila ada tamu aktif.
- `GET/POST/PUT/DELETE /penginapan/room-layouts[/:id]` — `location_id`, `room_id`, `row_number`, `urutan`; `GET` filter `location_id` (preload `location`, `room`).

### Guests (check-in / check-out / pindah)

`GET /penginapan/guests` — filter `status=inhouse|out`, `room_id`, `checkin_date` (limit 500). Setiap tamu disertai summary pembayaran:

```json
{ "id": 1, "room_id": 1, "name": "Budi", "harga": 100000,
  "room": { "room_number": "201" },
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

### Room histories

`GET /penginapan/room-histories` — filter `room_id`, `guest_id`, `status`.

---

## WebSocket

### GET `/ws?token=<JWT>`

Endpoint public untuk real-time update penginapan. Token JWT (hasil login `/auth/login`) dikirim sebagai query parameter; tanpa token atau token salah → `401` (respons HTTP sebelum upgrade). Client menerima push event setelah perubahan data di-commit; client tidak perlu mengirim pesan (ping/pong otomatis, ping tiap ±54 detik).

Format pesan (server → client):

```json
{"type": "room", "data": {"action": "status", "room": {"id": 1, "room_number": "201", "status": "maintenance", "current_occupants": 0, "location": {...}}}}
```

| `type` | `action` | `data` |
|--------|----------|--------|
| `room` | `create` / `update` / `status` / `occupancy` | `room` — objek kamar lengkap (preload `location`), berisi `current_occupants` terbaru |
| `room` | `delete` | `room_id` (int) |
| `guest` | `checkin` / `checkout` / `pindah` | `guest_ids` (array int) |

Event dikirim dari: `POST/PUT/DELETE /penginapan/rooms` (create/update/delete), `PUT /penginapan/rooms/:id/status` (status), check-in (occupancy per kamar + guest checkin), checkout, dan pindah kamar (guest pindah).

Contoh koneksi (test client resmi di `scripts/testing/wsclient`):

```bash
go run ./scripts/testing/wsclient "ws://127.0.0.1:8090/ws?token=$TOKEN" 15s
# CONNECTED
# EVENT type=room data={"action":"status","room":{...}}
```
