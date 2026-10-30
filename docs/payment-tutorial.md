# Panduan Penggunaan Payment Midtrans & Xendit

## 1. Konfigurasi Payment Gateway

Sebelum menggunakan fitur pembayaran, harus mengkonfigurasi gateway di sistem.

### 1.1 Tambah Konfigurasi Payment

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

Body request:

```json
{
  "key_configs": "payment_gateway",
  "value_configs": "midtrans"
}
```

Atau untuk xendit:

```json
{
  "key_configs": "payment_gateway",
  "value_configs": "xendit"
}
```

### 1.2 Konfigurasi Kredensial Midtrans

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "midtrans_server_key",
  "value_configs": "SB-xxx-xxx-xxx"  // Server key dari dashboard Midtrans
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "midtrans_env",
  "value_configs": "sandbox"  // atau "production"
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "midtrans_api_url",
  "value_configs": "https://api.sandbox.midtrans.com"  // atau production URL
}
```

### 1.3 Konfigurasi Kredensial Xendit

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "xendit_secret_key",
  "value_configs": "xch_test_..."  // Secret key dari dashboard Xendit
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "xendit_callback_token",
  "value_configs": "token-xendit-..."  // Token untuk verifikasi webhook
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "xendit_api_url",
  "value_configs": "https://api.xendit.co"  // Atau URL custom
}
```

### 1.4 Konfigurasi QRIS Statis

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "qris_payload",
  "value_configs": "00020101021126610014COM.GO-JEK.WWW01189360091432506719960210G1234567890303UMI51440014ID.CO.QRIS.WWW0215ID1020012345678900303UMI5204581253033605802ID5908TTPG PGN6007JAKARTA6105121106304ABCD"
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "qris_merchant",
  "value_configs": "Nama Hotel Anda"  // opsional, untuk tampilan di QR
}
```

```http
POST /penginapan/configs
Authorization: Bearer <JWT admin>
Content-Type: application/json
```

```json
{
  "key_configs": "nama_hotel",
  "value_configs": "Hotel Jaya"  // fallback: "PENGINAPAN"
}
```

## 2. Alur Penggunaan Payment

### 2.1 Check-in dengan Pembayaran

```http
POST /penginapan/guests
Authorization: Bearer <JWT>
```

Body:

```json
{
  "guests": [
    {"room_id": 1, "name": "Budi", "nik": "3175...", "gender": "pria", "harga": 100000}
  ],
  "payment": {
    "amount": 180000,
    "payment_method_id": 2,  // 2 = qris
    "source_id": 2,  // 2 = mesin (gateway)
    "status": "pending"
  }
}
```

Response `201`:

- `data.payment.status = "pending"` → tamu belum lunas, tampilkan QRIS
- `data.payment.status = "lunas"` → langsung lunas (amount cukup)

### 2.2 Charge QRIS via Gateway (Midtrans/Xendit)

```http
POST /penginapan/payments/:id/pay
Authorization: Bearer <JWT>
```

Body request (Midtrans):

```json
{
  "amount": 180000,
  "payment_method_id": 2,
  "source_id": 2
}
```

Body request (Xendit):

```json
{
  "amount": 180000
}
```

Response sukses:

```json
{
  "success": true,
  "data": {
    "payment_id": 3,
    "provider": "midtrans",
    "external_id": "TPG-PAY-3-1730000000",
    "qr_string": "00020101021126610014COM.GO-JEK.WWW...",
    "gateway_status": "pending",
    "expires_at": "2026-12-31 23:59:59"
  }
}
```

- `qr_string` ditampilkan ke tamu untuk di-scan
- `gateway_status = "pending"` hingga tamu lunas
- `expires_at` adalah waktu kadaluwarsa QRIS

### 2.3 Lunasi Pembayaran

#### 2.3.1 Melalui Webhook Gateway (Otomatis)

Setelah tamu scan QRIS dan gateway merespons, webhook akan dipanggil otomatis:

- **Midtrans**: `POST /penginapan/webhooks/midtrans` dengan body `{order_id, status_code, gross_amount, signature_key}`
- **Xendit**: `POST /penginapan/webhooks/xendit` dengan header `X-Callback-Token` dan body `{reference_id, status}`

Sistem secara otomatis:
1. Memverifikasi signature/token webhook
2. Update payment `status: "lunas"`, `paid_at: <now>`
3. Update tamu terkait `lunas` bila `total_bayar >= harga`
4. Kirim response `{"status": "ok", "payment_id": 3, "payment_status": "lunas"}`

#### 2.3.2 Manual via API

```http
PUT /penginapan/payments/3
Authorization: Bearer <JWT>
Body: {"status": "lunas"}
```

- System akan set `paid_at` dan tandai tamu sebagai lunas jika total cukup
- Jika amount kurang dari alokasi → `409 Conflict`

### 2.4 Void/ Batal Pembayaran

```http
DELETE /penginapan/payments/3
Authorization: Bearer <JWT>
```

- Status payment berubah menjadi `"batal"`
- Jika sudah batal → `409` (tidak bisa void dua kali)
- Kamar tidak otomatik dikosongkan (harus manual checkout)

### 2.5 Struk Pembayaran

```js
// Tampilkan struk di window baru
window.open('http://localhost:3080/penginapan/payments/3/receipt');

// Auto print
window.open('http://localhost:3080/penginapan/payments/3/receipt?autoprint=1');
```

- Public (tanpa auth), HTML 78mm siap cetak
- Nama hotel dari config `nama_hotel` (fallback: `PENGINAPAN`)
- Setiap tamu ada baris alokasi
- Status `batal` → cap `DIBATALKAN`
- `?autoprint=1` → langsung tampil dialog print

### 2.6 QRIS Statis (dari Config)

Jika sudah konfigurasi `qris_payload`:

```http
GET /penginapan/qris
Authorization: Bearer <JWT>
```

Response:

```json
{
  "data": {"qr_string": "00020101021126610014COM.GO-JEK.WWW..."}
}
```

- Gunakan saat check-in: tampilkan QR → tamu scan → `PUT /payments/:id {"status":"lunas"}`
- Belum dikonfigurasi → `404 NOT_CONFIGURED` dengan petunjuk

### 2.7 Cek Status Gateway

```http
GET /penginapan/payment-gateway
Authorization: Bearer <JWT>
```

Response:

```json
{
  "data": {
    "provider": "midtrans",
    "configured": true,
    "env": "sandbox",
    "webhook_path": "/penginapan/webhooks/midtrans"
  }
}
```

- `configured: true` bila semua kredensial terisi
- `configured: false` → sembunyikan opsi bayar QRIS di frontend

## 3. Troubleshooting

### 3.1 Masalah Umum

| Masalah | Penyebab | Solusi |
|---------|----------|--------|
| `404 NOT_CONFIGURED` | Gateway belum dikonfigurasi | Konfigurasikan via `POST /penginapan/configs` |
| `403 Amount mismatch` | Jumlah di webhook ≠ amount di payment | Pastikan `gross_amount` di webhook sama dengan `payment.amount` |
| `401 Invalid signature` | Signature webhook salah | Cek `SHA512(order_id + status_code + gross_amount + server_key)` untuk Midtrans |
| `401 X-Callback-Token invalid` | Token webhook Xendit salah | Cek header `X-Callback-Token` di dashboard Xendit |
| `404 Unknown external_id` | external_id tidak ditemukan di sistem | Pastikan external_id dari charge sesuai format `TPG-PAY-{id}-{timestamp}` |
| QRIS tidak muncul | `qris_payload` belum diisi | Tambahkan config `qris_payload` via `POST /penginapan/configs` |
| Struk tidak bisa dibuka | Bukan endpoint public | Pastikan request tanpa Authorization header |

### 3.2 Test Lingkungan

Gunakan script test di `scripts/testing/test_penginapan.sh`:

```bash
# 1. Login dulu
# 2. Tambah konfigurasi
POST /penginapan/configs ... payment_gateway midtrans
POST /penginapan/configs ... midtrans_server_key ...
POST /penginapan/configs ... midtrans_env sandbox
POST /penginapan/configs ... midtrans_api_url ...

# 3. Buat check-in dengan payment
POST /penginapan/guests ...

# 4. Charge QRIS
POST /penginapan/payments/{id}/pay ...

# 5. Simulasikan webhook Midtrans
POST /penginapan/webhooks/midtrans ...
```