# TTPG Digital Signage — Panduan Integrasi Client

Dokumen ini untuk tim yang membuat **player/client signage di luar repo ini**. Repo server hanya menyediakan API, admin UI, dan kontrak. Tidak ada kode web player, Electron, atau Android di repo ini.

Kontrak normatif lengkap ada di [`signage-client-contract.md`](signage-client-contract.md). Dokumen ini adalah ringkasan implementasi yang sudah aktif.

## 1. Ringkasan

- Server: satu instance, WebSocket hub in-memory. Tidak ada Redis/pub-sub.
- Device wajib punya `device_id` dan token per device.
- Device hanya menerima konten dari **snapshot published**. Draft tidak terlihat device.
- Semua endpoint device memakai header:
  - `X-Device-ID: <device_id>`
  - `X-Device-Token: <token>`
- Format response standar: `{success,message,data,pagination?,errors?}`.
- Base URL contoh: `https://signage.example.com`.
- Media publik ada di `/uploads/signage/...`; endpoint file device ada di `/signage/media/:id/file`.

## 2. Registrasi & Token

Admin membuat device di admin UI atau:

```http
POST /signage/devices
Authorization: Bearer <JWT admin>
Content-Type: application/json

{"device_id":"lobby-1","name":"Lobby","location":"Gedung A","status":"pending"}
```

Response membuat `data.device_token` **satu kali**. Simpan di secure storage device. Token di server hanya disimpan sebagai hash.

- Aktifkan device: `PUT /signage/devices/:id/settings` dengan `{"status":"active"}`.
- Rotasi token: `POST /signage/devices/:id/token/rotate` → token lama langsung tidak berlaku.
- Device nonaktif (`disabled`) → `403`.
- Token salah/tidak ada → `401`.

Jangan menaruh token di URL, log, screenshot, atau query string. Untuk WebSocket gunakan ticket sekali pakai.

## 3. Sync Manifest

```http
GET /signage/sync
X-Device-ID: lobby-1
X-Device-Token: <token>
If-None-Match: "sha256-<checksum>"   # opsional
```

Response sukses:

```json
{
  "success": true,
  "data": {
    "etag": "\"sha256-<checksum>\"",
    "manifest": {
      "version": 2,
      "generated_at": "2026-10-22T01:00:00Z",
      "server_time": "2026-10-22T01:00:00Z",
      "checksum": "<sha256>",
      "active": true,
      "device": {},
      "revision": {},
      "media": [],
      "layouts": [],
      "playlists": [],
      "schedules": [],
      "assignments": [],
      "tombstones": [],
      "commands": []
    }
  }
}
```

- `ETag` juga dikirim sebagai header. `304 Not Modified` = pakai cache lama; jangan reset playback.
- `active=false` = device belum disetujui/nonaktif. Tetap simpan manifest, tetapi tampilkan layar standby/fallback.
- `revision` = snapshot published yang aktif (bisa `null` bila belum ada publish).
- `media` hanya berisi file non-layout yang `enabled`.
- `layouts` berisi media `type=layout` + objek `layout`.
- `playlists[].items` berisi item urut `sort_order`.
- `schedules` dan `assignments` menentukan playlist yang diputar.
- `commands` berisi command pending/sent untuk device. Proses dan ack.
- `tombstones` berisi aset yang dihapus; hapus dari cache lokal.
- `device.settings` adalah objek bebas dari admin. `sync_interval` dan `telemetry_interval` dalam detik.

Algoritma client yang disarankan:

1. Simpan manifest baru secara atomik (tulis file/indexedDB sementara, lalu swap).
2. Jangan hapus cache lama sebelum manifest baru selesai diverifikasi.
3. Unduh media/layout yang checksum-nya belum ada di cache.
4. Hapus cache untuk ID di `tombstones` dan ID yang tidak lagi ada di manifest.
5. Jalankan scheduler memakai `server_time`, `device.timezone`, dan `schedules`.

## 4. Media & Cache

```http
GET /signage/media/:id/file
X-Device-ID: ...
X-Device-Token: ...
Range: bytes=0-1048575
```

- Server mendukung `Range`, `HEAD`, `ETag`, `X-Checksum`, `Accept-Ranges`.
- `checksum` adalah SHA-256 isi file.
- Ukuran file: `size_bytes`; MIME: `mime_type`; dimensi: `width`/`height` (jika tersedia).
- Website `type=website` tidak punya file; tampilkan online-only. Jika offline, gunakan `fallback` di `settings` atau placeholder.
- Layout `type=layout` tidak perlu diunduh sebagai file; render dari `layout`.
- Untuk video besar, cache per file dan verifikasi checksum sebelum dipakai.
- `cache_limit_mb` adalah batas yang disarankan. Client boleh melakukan eviction LRU, tetapi jangan menghapus media yang sedang diputar.

Lokasi cache yang disarankan:

- Web: Cache Storage + IndexedDB (metadata), fallback memory.
- Electron: folder `userData/cache/signage` + metadata JSON/SQLite.
- Android: internal storage + Room/SQLite; jangan simpan di external storage publik.

## 5. Scheduler & Playlist

Hari pada `days_mask`: bit 1 = Senin, bit 2 = Selasa, ..., bit 7 = Minggu. Contoh `127` = semua hari.

Aturan seleksi:

1. Kumpulkan `schedules` yang `enabled`, cocok `device_id` atau `group_id`, cocok tanggal, hari, dan jam.
2. Urutkan `priority` tertinggi.
3. Playlist dengan priority tertinggi menang. Jika beberapa sama, pilih deterministik berdasarkan `id` atau weight.
4. Jika tidak ada schedule aktif, gunakan `assignments` ke device, urut `priority_override` tertinggi.
5. Jika tetap kosong, tampilkan fallback/standby.

Jam:

- `start_time`/`end_time` format `HH:MM:SS` atau `HH:MM`.
- Jika `start_time <= end_time`: window di hari yang sama.
- Jika `start_time > end_time`: window melewati tengah malam.
- `start_date`/`end_date` inklusif.
- Gunakan timezone device, bukan timezone browser/server.

Playlist:

- `loop_mode=loop` ulang terus; `once` sekali lalu selesai.
- `repeat_count` ulang item/playlist.
- `selection_mode=sequential`: urut item; `weighted`: pilih acak berbobot `weight_percent`; `random`: acak.
- `duration_seconds` item menimpa durasi media; jika kosong pakai durasi media.
- Item bisa menunjuk layout (`layouts`) atau media biasa (`media`). Cari berdasarkan `media_id`.

### Layout utama (responsif)

Objek `layout` adalah kanvas desain utama. Jangan menempatkan elemen langsung berdasarkan ukuran layar device; render dulu kanvas utama, lalu skalakan:

```js
const L = layout; // {width, height, fit, letterbox, background, elements}
const fit = L.fit || "contain";
const lb = L.letterbox || "#000000";
const sx = screen.width / L.width;
const sy = screen.height / L.height;

let scaleX = sx, scaleY = sy;
if (fit === "contain") scaleX = scaleY = Math.min(sx, sy);
else if (fit === "cover") scaleX = scaleY = Math.max(sx, sy);
// "stretch": pakai sx/sy terpisah

const w = L.width * scaleX, h = L.height * scaleY;
const x = (screen.width - w) / 2, y = (screen.height - h) / 2;

// isi seluruh layar dengan warna letterbox, lalu gambar kanvas utama di x,y
// semua x/y/w/h/font_size/padding/radius elemen dikalikan scaleX/scaleY
```

- `contain` (default): seluruh layout terlihat, sisa ruang diwarnai `letterbox` (default hitam).
- `cover`: memenuhi layar dan memotong bagian berlebih.
- `stretch`: mengikuti lebar/tinggi layar; proporsi bisa berubah.
- `background`/`bg` bisa berupa `linear-gradient(...)`; render langsung sebagai CSS background. Abaikan metadata editor `*_mode`/`*2`/`*_angle`.
- Layout lama tanpa `fit`/`letterbox` diperlakukan sebagai `contain` + `#000000`.
- Untuk `website`, iframe tetap berada di dalam kanvas utama dan ikut diskalakan.

### Layout main + slot media

`manifest.device.main_layout_id` menunjuk layout yang dipakai sebagai kanvas saat playlist memutar media biasa:

```js
const main = manifest.layouts.find(l => l.id === manifest.device.main_layout_id);
const slot = main?.layout.elements
  .filter(e => e.type === "slot")
  .sort((a, b) => (a.slot === "main" ? -1 : 1) - (b.slot === "main" ? -1 : 1))[0];

if (main && slot) {
  renderLayout(main.layout, { media: currentMedia, slot }); // media masuk area slot
} else {
  renderFullscreen(currentMedia); // perilaku lama
}
```

- Hanya satu slot yang dipakai; layout lain/elemen non-slot menjadi overlay mengikuti `z`.
- Item playlist bertipe layout tetap dirender penuh; slot tidak diisi.
- `main_layout_id=null` atau layout/slot tidak ada → fallback media full screen.

## 6. Heartbeat & Telemetri

Kirim heartbeat setiap `device.telemetry_interval` detik (default 10 detik):

```http
POST /signage/device/heartbeat
X-Device-ID: lobby-1
X-Device-Token: <token>
Content-Type: application/json

{
  "app_version": "1.0.0",
  "config_version": "rev-123",
  "last_sync_at": "2026-10-22T01:00:00Z",
  "metrics": {
    "cpu_percent": 12.5,
    "memory_percent": 41.2,
    "disk_percent": 55.0,
    "source": "os"
  },
  "cache": {"bytes": 104857600, "items": 12},
  "now_playing": {"media_id": 7, "playlist_id": 2, "layout_id": 4},
  "screenshot_url": "/uploads/signage/screenshots/lobby-1_...png",
  "status": "ok"
}
```

- `source`: `os`, `browser-estimate`, atau `unknown`. Browser tidak bisa membaca RAM/HDD akurat; jangan memalsukan `os` untuk browser.
- Semua angka boleh `null` bila tidak tersedia.
- Server menyimpan sample per menit (retensi 7 hari) dan rollup 5 menit (90 hari).
- Online/offline dihitung server dari `last_seen_at` (2× interval telemetry), bukan dari field status.
- Supaya status online cepat terlihat, kirim heartbeat minimal tiap 10–30 detik.

## 7. Playback Report

Satu event per item selesai/dimulai. `event_id` wajib unik per device (ULID/UUID). Server idempoten: event dengan `event_id` sama tidak dicatat dua kali.

```http
POST /signage/device/playback
X-Device-ID: lobby-1
X-Device-Token: <token>
Content-Type: application/json

{
  "event_id": "01J...",
  "playlist_id": 2,
  "media_id": 7,
  "item_id": 11,
  "layout_id": 4,
  "revision_id": 3,
  "started_at": "2026-10-22T01:00:00Z",
  "ended_at": "2026-10-22T01:00:10Z",
  "duration_seconds": 10,
  "completed": true,
  "error": ""
}
```

- Kirim setelah item selesai atau gagal.
- Simpan antrean lokal saat offline; kirim ulang saat online. Server menolak `event_id` yang sudah dipakai device lain (`409`).
- `ended_at` dan `error` opsional; `duration_seconds` maksimal 86400.
- Endpoint batch lama `POST /signage/playback` tetap ada untuk kompatibilitas, tetapi client baru pakai `/signage/device/playback`.

## 8. Command & Ack

Admin bisa mengirim command lewat admin UI/API. Command muncul di manifest dan/atau WebSocket device.

```http
POST /signage/device/commands/:id/ack
X-Device-ID: lobby-1
X-Device-Token: <token>
Content-Type: application/json

{"status":"done","result":{"ok":true}}
```

Action yang dikenali server: `reload`, `sync_now`, `screenshot`, `restart_app`, `reboot`, `set_volume`, `set_brightness`, `update_config`, `custom`.

Dukungan per platform (rekomendasi):

| Action | Web | Electron | Android native |
|---|---|---|---|
| `reload` | `location.reload()` | `win.reload()` | restart activity/WebView |
| `sync_now` | sync manifest | sync manifest | sync manifest |
| `screenshot` | `canvas`/`html2canvas` bila tersedia, lalu upload | `capturePage()` | MediaProjection (izin khusus) |
| `restart_app` | halaman reload | `app.relaunch()` | `ProcessPhoenix`/restart service |
| `reboot` | tidak didukung | OS permission/`shutdown` | device owner/root |
| `set_volume` | tidak didukung | `webContents.setAudioMuted`/OS | `AudioManager` |
| `set_brightness` | tidak didukung | OS API | `Settings.System`/device owner |
| `update_config` | terapkan settings | terapkan settings | terapkan settings |
| `custom` | tangani sendiri | tangani sendiri | tangani sendiri |

Semua ack harus mengirim `status` `done` atau `failed`. Jika action tidak didukung, ack `failed` dengan alasan.

## 9. Screenshot

```http
POST /signage/device/screenshot
X-Device-ID: lobby-1
X-Device-Token: <token>
Content-Type: multipart/form-data

file=@screenshot.png
```

- JPG/PNG/WebP, maks 5MB.
- Response berisi `screenshot_url`.
- Screenshot bisa juga dikirim lewat heartbeat (`screenshot_url`) bila sudah diunggah.
- Admin melihat screenshot di tab Monitoring.

## 10. WebSocket

### Admin (JWT role admin)

```text
wss://host/signage/monitor/ws?token=<JWT>
```

Event yang mungkin diterima: `telemetry`, `device_status`, `device_ws`, `command`, `playback`, `screenshot`, `publish`, `media`, `layout`.

Format:

```json
{"type":"command","data":{"device_id":"lobby-1","command_id":12,"status":"done"}}
```

### Device (ticket sekali pakai)

1. `POST /signage/device/ws-ticket` dengan header device → `{ticket, expires_in:60}`.
2. Connect `wss://host/signage/device/ws?ticket=<ticket>`.
3. Ticket hanya berlaku 60 detik dan sekali pakai.

Event server ke device:

```json
{"type":"command","data":{"id":12,"device_id":"lobby-1","action":"reload","payload_json":"","expires_at":"..."}}
{"type":"refresh","data":{"revision_id":3}}
```

Device boleh mengirim:

```json
{"type":"heartbeat"}
{"type":"ack","command_id":12,"status":"done","result":{"ok":true}}
```

Fallback: jika WS putus, tetap lakukan sync/heartbeat HTTP sesuai interval. Reconnect dengan exponential backoff + jitter. Jangan reconnect loop tanpa jeda.

## 11. Error & Retry

- `401`: token/ticket salah atau device tidak terdaftar. Jangan retry cepat; periksa provisioning.
- `403`: device disabled.
- `404`: media/revision/command tidak ada.
- `409`: konflik (mis. event_id dipakai device lain).
- `5xx`: retry dengan backoff.
- `304`: bukan error; cache valid.

Aturan retry yang disarankan:

- Sync: 60 detik (atau `device.sync_interval`), backoff 5s→10s→30s→60s saat gagal.
- Heartbeat: 10–30 detik, jangan menumpuk request.
- Playback: antrean lokal, retry sampai sukses.
- Media: resume dengan `Range` bila koneksi putus.

## 12. Keamanan

- Simpan token di secure storage: Android Keystore, Electron `safeStorage`, macOS Keychain, Windows DPAPI.
- Jangan log token, ticket, atau isi file.
- Jangan menaruh token di URL publik; WebSocket device memakai ticket.
- Validasi checksum media sebelum render.
- Untuk website: whitelist domain di settings bila perlu; website online-only.
- Android disarankan device owner + lock task mode untuk kiosk. WebView harus mengizinkan media autoplay dan fullscreen sesuai kebutuhan.

## 13. Rekomendasi Platform

- **Web player (browser kiosk)**: JavaScript tanpa build step, Cache Storage/IndexedDB, Service Worker opsional. Batasan: screenshot/volume/brightness/reboot terbatas; metric `browser-estimate`.
- **Electron (Linux/Windows)**: pilihan default untuk kiosk desktop. Akses filesystem/OS lebih lengkap, cache stabil, auto-launch, `capturePage`, `app.relaunch`.
- **Android native Kotlin**: preferensi untuk TV box/tablet. `minSdk 26`, package `id.ttpg.signage`, device owner/lock task, WebView untuk layout website, foreground service untuk sync/heartbeat, Room untuk cache. Screenshot butuh MediaProjection/izin device owner.
- **Android WebView-only**: cukup untuk konten sederhana, tetapi command OS dan offline cache terbatas.

## 14. Checklist Uji Client

- [ ] Provisioning token + rotasi token.
- [ ] Sync pertama, sync `304`, sync setelah publish baru.
- [ ] Cache media + verifikasi checksum + Range/resume.
- [ ] Layout teks/gambar/video/website/jam/ticker/shape tampil benar.
- [ ] Scheduler: hari, jam, overnight, tanggal, priority, group.
- [ ] Playlist loop/once/repeat/weighted.
- [ ] Offline: cache tetap jalan, heartbeat/playback antre.
- [ ] Online kembali: sync, kirim antrean playback, tidak duplikat.
- [ ] Command reload/sync_now/screenshot + ack done/failed.
- [ ] WebSocket device ticket + reconnect; admin monitor menerima telemetry/command.
- [ ] Tombstone menghapus cache.
- [ ] Device disabled → berhenti menampilkan konten aktif.
