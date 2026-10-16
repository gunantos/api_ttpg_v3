# API Penginapan — Spesifikasi Frontend

Dokumen ringkas untuk membangun UI modul penginapan. Base URL server: `http://<host>:3080` (contoh LAN: `http://192.168.110.100:3080`). Dokumen lengkap interaktif: buka `GET /docs` di browser.

## Autentikasi

Semua endpoint di bawah ini **butuh JWT** (kecuali struk, diberi tahu di tempatnya):

```bash
# 1. login → token (berlaku 7 hari)
POST /auth   {"username":"...","password":"..."}
→ {"success":true,"data":{"token":"<JWT>","user_id":1,...}}

# 2. pakai di semua request
Authorization: Bearer <JWT>
```

## Format respons

Sukses: `{ "success": true, "message": "...", "data": ... }` (ada `pagination` bila list terpaginasi).
Gagal: `{ "success": false, "message": "...", "errors":[...] }` — cek `message` untuk ditampilkan ke user.

Kode HTTP yang akan Anda temui: `400` data tidak valid · `401` token salah/kedaluwarsa · `404` tidak ada · `409` bentrok aturan bisnis (lihat [Aturan & jebakan](#aturan--jebakan)).

## Daftar endpoint

| Method | Path | Keterangan |
|---|---|---|
| GET | `/penginapan/payment-methods` | Opsi metode bayar (dropdown) |
| GET | `/penginapan/purchase-sources` | Opsi sumber pembelian (dropdown) |
| GET | `/penginapan/locations` | Master lokasi |
| GET/POST/PUT/DELETE | `/penginapan/locations[/:id]` | CRUD lokasi |
| GET | `/penginapan/rooms` | List kamar + **`available`** |
| GET/POST/PUT/DELETE | `/penginapan/rooms[/:id]` | CRUD kamar |
| PUT | `/penginapan/rooms/:id/status` | Set `kosong`/`maintenance` |
| GET | `/penginapan/room-layouts?location_id=` | Layout tampilan per lokasi |
| GET | `/penginapan/guests` | List tamu + ringkasan bayar |
| GET | `/penginapan/guests/:id` | Detail tamu |
| POST | `/penginapan/guests` | **Check-in** (1+ tamu, bisa sambil bayar) |
| PUT | `/penginapan/guests/:id` | Update nama/nik/gender/harga |
| POST | `/penginapan/guests/:id/checkout` | Check-out |
| POST | `/penginapan/guests/:id/pindah` | Pindah kamar |
| GET | `/penginapan/guests/:id/histories` | History tamu (check-in/out/pindah) |
| POST | `/penginapan/guests/:id/ktp` | Upload foto KTP (multipart `file`, maks 5MB) |
| GET | `/penginapan/guests/:id/ktp` | Ambil foto KTP (bytes gambar, perlu auth) |
| DELETE | `/penginapan/guests/:id/ktp` | Hapus foto KTP |
| GET | `/penginapan/room-histories?room_id=` | History per kamar |
| GET | `/penginapan/payments` | List pembayaran (`?guest_id=`, `?status=`) |
| GET/PUT/DELETE | `/penginapan/payments/:id` | Detail / update / void |
| POST | `/penginapan/payments` | Bayar manual (di luar check-in) |
| POST | `/penginapan/payments/:id/pay` | Charge QRIS via gateway (bila dikonfigurasi) |
| GET | `/penginapan/qris` | QRIS statis dari config |
| GET | `/penginapan/payment-gateway` | Info gateway aktif (frontend cek dulu) |
| GET | `/penginapan/payments/:id/receipt` | **Struk HTML — tanpa auth**, untuk `window.open` |

---

## 1. Dropdown & master

```http
GET /penginapan/payment-methods   → data: [{id, code:"cash", name:"Tunai", ...}]
GET /penginapan/purchase-sources  → data: [{id, code:"langsung", name:"Langsung", ...}]
GET /penginapan/locations         → data: [{id, name, keterangan}]
```

Lokasi: `POST/PUT {name, keterangan}` — **keduanya wajib** (400 bila salah satu kosong).

## 2. Kamar & ketersediaan

```http
GET /penginapan/rooms
GET /penginapan/rooms?status=kosong&location_id=1&room_type=pria
```

```json
{ "success": true, "data": [{
    "id": 1, "location_id": 1,
    "room_number": "201", "room_type": "pria",
    "capacity": 2, "current_occupants": 1,
    "status": "terisi", "available": false,
    "location": {"id": 1, "name": "Gedung A"} }] }
```

**`available` (field turunan — baca ini untuk ketersediaan):** `true` bila `status == "kosong"` dan `current_occupants < capacity`. Field ini ada di **semua** objek kamar (list, detail, CRUD, `room` di response tamu, dan event WebSocket).

**Aturan UI pilih kamar (check-in):** tampilkan **semua** kamar — jangan disaring/d disembunyikan. Kamar dengan `room_type` ≠ gender tamu yang sedang diisi → tampil **disabled**: **bukan warna hijau**, tidak bisa diklik/dipilih. Hijau + bisa diklik hanya bila `room_type` **sama** dengan gender tamu **dan** `available:true` (kamar beda gender yang available juga tetap disabled). Query `?room_type=` hanya untuk keperluan lain — jangan dipakai untuk menyembunyikan kamar dari layar pemilihan.

Buat kamar: `POST {location_id?, room_number, room_type:"pria"|"wanita", capacity?}` — `(location_id, room_number)` unik (kelangkaan → error). Layout: `POST {location_id?, room_id?, row_number, urutan}`.

## 3. Check-in

```http
POST /penginapan/guests
```

```json
{
  "guests": [
    {"room_id": 1, "name": "Budi", "nik": "3175...", "gender": "pria", "harga": 100000,
     "checkin_date": "2026-10-16", "checkin_time": "14:00"}
  ],
  "payment": {
    "amount": 100000, "payment_method_id": 1, "source_id": 1,
    "status": "lunas"
  }
}
```

- `guests`: minimal 1. `room_id`, `name`, `gender` (`pria`/`wanita`) wajib; `nik`, `harga`, `checkin_date`, `checkin_time` opsional (default harga 0; tanggal/jam default sekarang).
- `payment` **opsional**. `status` default `lunas`; `"pending"` untuk bayar nanti. `amount` wajib = total yang dibayar.
- **2 tamu+ dalam 1 payment** → `amount` = Σ tagihan dan wajib isi `allocations`:

```json
"payment": {
  "amount": 180000, "payment_method_id": 2, "source_id": 2, "status": "pending",
  "allocations": [
    {"guest_index": 0, "amount": 90000},
    {"guest_index": 1, "amount": 90000}
  ]
}
```

(`guest_index` = urutan di array `guests`. 1 tamu → alokasi otomatis penuh, allocations tidak perlu.)

Respons `201`: `data.guests[]` (tamu + `total_bayar`, `sisa`, `lunas`) dan `data.payment` (`id`, `status`, `paid_at`, `source`).

## 4. Daftar & detail tamu

```http
GET /penginapan/guests?status=inhouse          # inhouse | out
GET /penginapan/guests?room_id=1&checkin_date=2026-10-16
```

Maks 500 baris. Setiap tamu memuat `room` (lengkap, termasuk `available`) + ringkasan pembayaran:

```json
{ "id": 1, "room_id": 1, "name": "Budi", "nik": "3175...", "ktp_photo": "1_1760..._a1b2c3d4.png",
  "gender": "pria", "harga": 100000,
  "checkin_date": "2026-10-16", "checkin_time": "14:00:00", "checkout_date": null,
  "room": {"room_number": "201", "status": "terisi", "available": false},
  "total_bayar": 100000, "sisa": 0, "lunas": true }
```

- `total_bayar` = Σ pembayaran berstatus `lunas` untuk tamu itu; `sisa` = `max(0, harga - total_bayar)`; `lunas` = `total_bayar >= harga`.
- Update tarif: `PUT /guests/:id {"harga":120000}` → `sisa`/`lunas` dihitung ulang otomatis.

## 5. Check-out, pindah, history

```http
POST /penginapan/guests/1/checkout        # body: {}
POST /penginapan/guests/1/pindah          # body: {"room_id": 4}
GET  /penginapan/guests/1/histories
GET  /penginapan/room-histories?room_id=1
```

- Check-out: kosongkan `checkin` → kamar otomatis `kosong`. Sudah pernah checkout → **409**.
- Pindah: tamu tetap (tidak ada baris baru), `room_id` pindah + history `pindah`. Kamar sama → 400.
- History: `[{"type":"check-in"|"check-out"|"pindah", ...}]`.

## 6. Pembayaran

**Bayar manual** (tanpa check-in baru):

```http
POST /penginapan/payments
{"guest_ids":[1], "amount":50000, "payment_method_id":1, "source_id":1}
```

1 tamu → alokasi otomatis; multi tamu → wajib `allocations` pakai `guest_id` (bukan `guest_index`): `[{"guest_id":1,"amount":50000}]`.

**Lunasi pembayaran pending** (mis. tamu bayar via mesin/QRIS):

```http
PUT /penginapan/payments/3   {"status":"lunas"}     # paid_at terisi; tamu terkait otomatis lunas bila total cukup
PUT /penginapan/payments/3   {"amount":20000}       # turunkan nominal (409 bila < total alokasi)
DELETE /penginapan/payments/3                       # void → status "batal" (409 bila sudah batal)
```

List pembayaran: `GET /penginapan/payments?guest_id=1&status=lunas`.

## 7. Struk, QRIS, gateway

**Struk** — HTML 78mm siap cetak, **public (tanpa token)**:

```js
window.open(`${BASE}/penginapan/payments/3/receipt`);              // tampil
window.open(`${BASE}/penginapan/payments/3/receipt?autoprint=1`);  // auto print dialog
```

**QRIS statis** (bila hotel sudah set config `qris_payload`):

```http
GET /penginapan/qris   → {"data":{"qr_string":"000201..."}}
```
Belum dikonfigurasi → `404` `NOT_CONFIGURED`.

**Payment gateway otomatis (Midtrans/Xendit)** — cek dulu:

```http
GET /penginapan/payment-gateway            → {"data":{"provider":"midtrans","configured":true}}
POST /penginapan/payments/3/pay            → {"data":{"external_id":"...","qr_string":"...","expires_at":"..."}}
```

Alur frontend: check-in dengan `payment.status=pending` → tampilkan QR (`qr_string` dari `/qris` statis **atau** hasil `POST /payments/:id/pay`) → tunggu admin lunasi (`PUT .../status lunas`) atau webhook gateway (otomatis) → tampilkan struk. `configured:false` → sembunyikan opsi bayar QRIS. Charge ulang payment yang sudah lunas → 409 `PAYMENT_PAID`.

## 8. Foto KTP (opsional)

Upload setelah check-in (guest id sudah ada). Field `ktp_photo` di response tamu = nama file, `null` = belum ada.

```http
POST /penginapan/guests/:id/ktp        # multipart form-field "file", JPG/PNG/WebP, maks 5MB
GET  /penginapan/guests/:id/ktp        # bytes gambar (Content-Type: image/*)
DELETE /penginapan/guests/:id/ktp
```

```bash
curl -X POST "$BASE/penginapan/guests/1/ktp" \
  -H "Authorization: Bearer $JWT" \
  -F "file=@/path/foto_ktp.jpg;type=image/jpeg"
# → {"success":true,"data":{"guest_id":1,"ktp_photo":"1_1760..._a1b2c3d4.png"}}
```

- Upload ulang **mengganti** foto lama (file lama dihapus otomatis).
- **Tampilkan gambar**: `<img>` tidak bisa kirim header auth → fetch sebagai blob dulu:

```js
const res = await fetch(`${BASE}/penginapan/guests/${id}/ktp`, { headers: { Authorization: `Bearer ${token}` } });
if (res.ok) img.src = URL.createObjectURL(await res.blob());   // 404 = belum ada foto
```

- Error: 400 bukan gambar/tanpa file/kebesaran · 404 tamu belum ada foto atau file hilang dari disk.

## 9. WebSocket (update real-time)

```js
const ws = new WebSocket(`ws://${host}/ws?token=${JWT}`);   // auth via query
```

| `type` | `action` | `data` |
|---|---|---|
| `room` | `create` / `update` / `status` / `occupancy` | `room` — objek kamar terbaru (termasuk `available`) |
| `room` | `delete` | `room_id` |
| `guest` | `checkin` / `checkout` / `pindah` | `guest_ids: [1, 2]` |

Setelah event `room` → refresh panel kamar; setelah `guest` → refresh daftar tamu + ringkasan.

---

## Aturan & jebakan

1. **`lunas`** = total pembayaran berstatus `lunas` ≥ `harga` tamu. Pembayaran `pending` tidak dihitung. Satu payment bisa menutup banyak tamu (via allocations).
2. **`available`** di kamar = kamar **benar-benar kosong** (`status=="kosong"`). Kamar `terisi` 1 dari 2 tempat tidur tetap `available:false` (definisi bisnis); check-in menolak hanya bila **penuh** atau **maintenance**.
3. **409 check-in**: kamar penuh — `"kamar X tidak mencukupi (sisa kapasitas N)"`; kamar maintenance — `"kamar X dalam maintenance"`. 404: `room_id` tidak ada.
4. **409 lain**: checkout ganda · pindah ke kamar sama (400) · void payment sudah batal · `PUT` amount < total alokasi · delete kamar masih ada tamu aktif.
5. **Time**: request check-in menerima `checkin_date` `"YYYY-MM-DD"` dan `checkin_time` `"HH:MM"`; response memakai `"HH:MM:SS"` + timestamp RFC3339 (`created_at` dst).
6. **Nominal** = integer rupiah (tanpa desimal). `amount` pembayaran saat check-in wajib ≥ 1.
7. List tamu maks 500 baris (filter dulu per `room_id`/`status`/`checkin_date` bila perlu). List rooms/payments tanpa paginasi.
8. **Foto KTP** disimpan di server (`$UPLOAD_DIR/ktp`, default `./uploads/ktp` relatif terhadap CWD server) dengan nama file dibuat server (`<guest_id>_<ts>_<rand>.<ext>`) — nama dari client diabaikan. Ganti foto = file lama dihapus. `GET .../ktp` butuh JWT (KTP = data sensitif); tampilkan via fetch-blob, bukan `<img src>` polos.
9. **Hold pembayaran cash**: payment `pending` metode cash (tanpa gateway) otomatis **batal** + tamu yang belum lunas di-check-out otomatis setelah `payment_hold_minutes` (**default 15 menit**; config via `POST /penginapan/configs`, `0` = nonaktif). Sweep tiap 60 detik → kamar langsung `available` + event WS. Frontend boleh tampilkan countdown, tapi jangan mengandalkan status `pending` yang lama; refresh setelah event `guest`/`room`.
