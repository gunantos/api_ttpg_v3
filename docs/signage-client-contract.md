# Kontrak Client Digital Signage v2

> **STATUS: AKTIF.**
> Backend, migration `012`–`016`, WebSocket, dan admin UI basic sudah diimplementasikan.
> Dokumen ini adalah kontrak untuk client digital signage eksternal; daftar endpoint ringkas ada di [`docs/api.md`](api.md#digital-signage-v2--aktif).
> Panduan langkah-demi-langkah: [`signage-client-readme.md`](signage-client-readme.md).

## 1. Keputusan arsitektur

- Client/player dibuat di luar repo ini; repo menyediakan API, kontrak, dan dokumentasi cache.
- Autentikasi device memakai **token per device** (`X-Device-ID` + `X-Device-Token`).
- Perubahan konten memakai alur **draft → publish → rollback**.
- Website bersifat **online-only**; saat offline client memakai `fallback_url` atau placeholder.
- Monitoring memakai **WebSocket dua arah** untuk telemetry dan remote command, dengan fallback HTTP.
- Metrik hardware bersifat **opsional/fleksibel**: client mengirim yang tersedia, server menampilkan `unknown` bila tidak ada.
- Retensi metrik: **raw 1 menit selama 7 hari**, **rollup 5 menit selama 90 hari**.

## 2. Lifecycle device

1. Admin membuat device di `/admin` → server membuat `device_id` + token sekali tampil.
2. Admin menyalin setup (`base_url`, `device_id`, `device_token`) ke client, atau memindai QR setup.
3. Client menyimpan kredensial secara lokal dan melakukan sync pertama.
4. Client mengunduh manifest + aset, memverifikasi checksum, lalu mulai playback.
5. Client mengirim heartbeat/telemetry dan menerima command.
6. Token dapat di-rotate/revoke kapan saja dari admin.

Status device:

| Status | Arti |
|---|---|
| `pending` | Terdaftar, belum diaktifkan; hanya boleh heartbeat/sync terbatas. |
| `active` | Boleh sync penuh, playback, dan menerima command. |
| `disabled` | Token ditolak; client harus menampilkan layar nonaktif. |

Online/offline **bukan** status admin. Online dihitung dari `last_seen_at` dan koneksi WebSocket.

## 3. Autentikasi

### Admin
- JWT/API key yang sudah ada (`auth.AuthMiddleware`).
- Endpoint admin wajib role `admin`.

### Device HTTP
```http
X-Device-ID: screen-lobby-1
X-Device-Token: <token>
```

Token disimpan di server sebagai hash (mis. SHA-256), dibandingkan constant-time.
Token asli hanya ditampilkan sekali saat dibuat/rotate.

### Device WebSocket
1. `POST /signage/device/ws-ticket` dengan header device → `{ticket, expires_in}`.
2. `GET /signage/device/ws?ticket=<ticket>` → upgrade WebSocket.
3. Ticket sekali pakai, berlaku ±60 detik, tidak boleh masuk log.

### Admin WebSocket
- `GET /signage/monitor/ws?token=<JWT>` (pola sama dengan `/ws` penginapan).
- Hanya role `admin`.
- Initial snapshot dikirim setelah koneksi; update berikutnya berupa event.

## 4. Daftar endpoint

### Device runtime (token device)
| Method | Path | Fungsi |
|---|---|---|
| GET | `/signage/sync` | Manifest terbaru; ETag/304; device settings + konten published. |
| GET | `/signage/media/:id/file` | Download media; Range, ETag, HEAD, checksum. |
| POST | `/signage/device/ws-ticket` | Ticket WebSocket sekali pakai. |
| GET | `/signage/device/ws` | Kanal telemetry naik + command turun. |
| POST | `/signage/device/heartbeat` | Fallback HTTP: telemetry + ambil command + server time. |
| POST | `/signage/device/commands/:id/ack` | Ack command via HTTP (fallback). |
| POST | `/signage/device/screenshot` | Upload screenshot layar. |
| POST | `/signage/device/playback` | Batch proof-of-play idempoten. |

### Admin (JWT/API key role admin)
| Method | Path | Fungsi |
|---|---|---|
| POST | `/signage/devices/:id/token/rotate` | Rotate token, respons sekali tampil. |
| PUT | `/signage/devices/:id/settings` | Timezone, volume, brightness, cache limit, dll. |
| GET | `/signage/devices/:id/overview` | Status, metrik terakhir, now playing, alert, cache. |
| GET | `/signage/devices/:id/metrics?from=&to=&resolution=raw|5m` | Grafik metrik. |
| GET | `/signage/devices/:id/events` | Event online/offline/error/command. |
| POST | `/signage/commands` | Kirim command `{device_id,action,payload,expires_seconds}`. |
| GET | `/signage/commands?device_id=&status=` | Riwayat command. |
| POST | `/signage/publish/preview` | Preview draft (bukan published); body scope opsional. |
| POST | `/signage/publish` | Publish snapshot ke device/grup/semua. |
| GET | `/signage/publish/history` | Riwayat revision. |
| POST | `/signage/publish/:revision_id/rollback` | Rollback device/grup ke revision lama. |
| GET | `/signage/revisions` / `/signage/revisions/:id` | Riwayat/detail revision. |
| GET | `/signage/snapshot` | Export snapshot draft. |
| GET | `/signage/schedule/resolve?device_id=` | Resolusi jadwal aktif server-side. |
| GET | `/signage/monitor/summary` | Ringkasan online/offline/alert. |
| GET | `/signage/monitor/ws?token=` | Stream monitoring admin. |

### Kompatibilitas
- Endpoint lama `/signage/config`, `/signage/device/status`, `/signage/playback` tetap dipertahankan minimal satu siklus rilis.
- Client baru wajib memakai `/signage/sync`.
- Respons lama tidak diubah agar client lama tidak rusak.

## 5. Manifest sync

`GET /signage/sync` mengembalikan manifest **published** untuk device yang terautentikasi.

Header:
- `ETag: "<sha256>"`
- `Cache-Control: no-cache` (wajib revalidate)
- `Vary: X-Device-ID`

Jika `If-None-Match` sama → `304 Not Modified`, body kosong.
Jika device belum punya revision published → manifest memakai data live (mode bootstrap/kompatibilitas). Setelah publish pertama, device hanya menerima snapshot published.

Contoh ringkas:

```json
{
  "success": true,
  "message": "Manifest signage",
  "data": {
    "etag": "\"sha256-abc123\"",
    "manifest": {
      "version": 2,
      "generated_at": "2026-10-21T10:00:00Z",
      "checksum": "abc123",
      "active": true,
      "server_time": "2026-10-21T10:00:00Z",
      "device": {
        "device_id": "screen-lobby-1",
        "name": "Lobby Utama",
        "timezone": "Asia/Jakarta",
        "volume": 80,
        "brightness": 80,
        "orientation": "landscape",
        "resolution": "1920x1080",
        "cache_limit_mb": 2048,
        "config_version": "v1",
        "settings": {},
        "sync_interval": 60,
        "telemetry_interval": 10
      },
      "revision": {"id": 12, "version": "rev-12", "checksum": "def456"},
      "media": [
        {
          "id": 17,
          "type": "image",
          "name": "Promo Pagi",
          "url": "/uploads/signage/promo.png",
          "checksum": "sha256:abc",
          "size_bytes": 184320,
          "mime_type": "image/png",
          "width": 1920,
          "height": 1080,
          "duration_seconds": 10
        }
      ],
      "layouts": [
        {"id": 2, "name": "Welcome", "duration_seconds": 15, "checksum": "sha256:def", "layout": {"width": 1920, "height": 1080, "elements": []}}
      ],
      "playlists": [
        {"id": 3, "name": "Pagi", "priority": 10, "loop_mode": "loop", "selection_mode": "sequential", "items": [{"id": 9, "media_id": 17, "type": "image", "duration_seconds": 10, "sort_order": 1}]}
      ],
      "schedules": [
        {"id": 4, "playlist_id": 3, "device_id": "screen-lobby-1", "days": [1, 2, 3, 4, 5], "start_time": "06:00:00", "end_time": "10:00:00", "priority": 100}
      ],
      "assignments": [
        {"id": 5, "device_id": "screen-lobby-1", "playlist_id": 3}
      ],
      "tombstones": [
        {"entity": "media", "entity_id": 8, "deleted_at": "2026-10-20T09:00:00Z"}
      ],
      "commands": []
    }
  }
}
```

### Aturan manifest
- `media`, `layouts`, `playlists`, `schedules` adalah **full snapshot**, bukan delta.
- `tombstones` membantu client menghapus cache lama.
- `active` dihitung server; client tetap menyimpan `schedules` untuk evaluasi offline.
- `checksum` wajib untuk media dan layout; client wajib memverifikasi.
- `size_bytes` untuk estimasi cache dan progress download.
- `download_url` boleh berupa signed URL; jika tidak, media publik boleh memakai `url`.
- `server_time` untuk koreksi jam client.

## 6. Cache & offline

### Storage
- Metadata manifest: **IndexedDB**.
- File media: **Cache Storage**.
- Key cache: `signage/media/<checksum>` (content-addressed).
- Simpan juga: manifest JSON, `server_time_offset`, event playback yang belum terkirim.

### Sinkronisasi
1. Baca manifest cache saat boot; render segera walau offline.
2. `GET /signage/sync` dengan `If-None-Match`.
3. `200` → validasi schema; `304` → tetap.
4. Bandingkan media/layout yang berubah.
5. Download aset baru dengan concurrency maksimal 3, retry 3× exponential backoff.
6. Verifikasi checksum SHA-256.
7. Setelah semua aset playlist aktif siap → atomic swap manifest.
8. Preload item berikutnya.
9. Garbage collect cache yang tidak direferensikan (kecuali aset aktif).
10. Ulangi tiap `sync_interval_seconds` (default 60 detik).

### Quota & eviction
- Hormati `cache_limit_mb` dari device.
- LRU, tetapi jangan hapus:
  - item yang sedang diputar,
  - aset manifest aktif,
  - item berikutnya yang sudah di-preload.
- Jika cache penuh: hapus manifest lama terlebih dahulu, lalu aset orphan.
- Jika aset aktif gagal diunduh: tetap pakai versi cache lama; kirim event `download_failed`.

### Offline
- Tetap putar manifest terakhir.
- Jadwal dihitung lokal memakai:
  - `timezone` device,
  - `server_time_offset` terakhir,
  - `schedules` + `playlists` dari manifest.
- Website: jangan dicache; tampilkan `fallback_url` atau placeholder.
- Jam/ticker tetap berjalan dari waktu lokal.
- Playback log disimpan antre dan dikirim saat online.
- Heartbeat tetap dicoba dengan backoff; status server menjadi offline setelah 2× interval.

### Error state
Client harus bisa menampilkan overlay/status:
- `no_manifest` — belum pernah sync.
- `stale_manifest` — manifest lama, server tidak terjangkau.
- `cache_full` — quota tidak cukup.
- `asset_missing` — checksum/file gagal.
- `invalid_token` — token ditolak; butuh setup ulang.

## 7. Algoritma jadwal & playlist

Server menghitung `active`; client wajib bisa menghitung ulang saat offline.

Urutan:
1. Filter schedule `enabled`.
2. Cocokkan target: device, grup, atau global.
3. Cocokkan `start_date`/`end_date`.
4. Cocokkan `days_mask`/`days`.
5. Cocokkan jam, termasuk jadwal lintas tengah malam (`22:00–02:00`).
6. Urutkan: `schedule.priority` desc → `playlist.priority` desc → assignment priority desc → id asc.
7. Schedule pertama yang cocok menang.
8. Jika tidak ada schedule: pakai playlist yang di-assign, urut priority/override.
9. Jika tidak ada assignment: pakai `default_playlist_id` dari config device/global.
10. Jika tetap kosong: tampilkan fallback screen.

Playlist:
- `loop_mode=loop` → ulang terus.
- `loop_mode=once` → sekali lalu lanjut playlist berikutnya/fallback.
- `repeat_count` → jumlah pengulangan playlist.
- `selection_mode=sequential` → item berurutan.
- `selection_mode=weighted` → pilih item berdasarkan `weight_percent`.
- `duration_seconds` item menimpa durasi media.
- Item `enabled=false` dilewati.

## 8. Layout schema

Schema saat ini tetap didukung. Target v2 menambah field opsional berikut.

### Layout utama & responsif
Objek `layout` adalah **kanvas desain utama**:

```json
{
  "width": 1920,
  "height": 1080,
  "background": "#0f172a",
  "fit": "contain",
  "letterbox": "#000000",
  "elements": []
}
```

- `width`/`height` = resolusi desain. Semua koordinat elemen memakai piksel desain ini.
- `fit=contain` (default) menskalakan seluruh kanvas dengan `min(screen_w/width, screen_h/height)`, menengahkan, dan mengisi sisa layar dengan `letterbox` (default `#000000`).
- `fit=cover` memakai `max(...)` lalu crop bagian berlebih.
- `fit=stretch` menskalakan sumbu X/Y terpisah.
- Kalikan `x/y/w/h`, `font_size`, `padding`, `radius`, dan `border` dengan skala yang sama.
- Layout lama tanpa `fit`/`letterbox` wajib diperlakukan sebagai `contain` + `#000000`.
- `background` (layout) dan `bg` (elemen) dapat berupa warna solid atau CSS `linear-gradient(...)`. Field `background_mode`/`background2`/`background_angle` dan `bg_mode`/`bg2`/`bg_angle` hanya metadata editor; client cukup memakai `background`/`bg`.

### Layout main device & slot media
- Manifest memuat `device.main_layout_id` (nullable). Layout tersebut ada di `layouts`.
- Device memakai layout main saat item playlist adalah **media biasa** (gambar/video/website). Media dirender ke elemen `type=slot` (utamakan `slot="main"`).
- Satu layout maksimal satu slot efektif. Jika ada beberapa, pakai slot pertama `slot="main"`, lalu slot pertama bertipe `slot`.
- Jika `main_layout_id=null`, layout tidak ditemukan, atau layout tidak punya slot: render media full screen (perilaku lama).
- Jika item playlist adalah layout, render layout tersebut sebagai layout penuh; slot tidak diisi media playlist.
- Elemen non-slot dirender sebagai overlay mengikuti `z`, termasuk di atas/bawah slot.

### Umum per elemen
```json
{
  "id": "el1",
  "type": "text",
  "name": "Judul",
  "x": 100,
  "y": 100,
  "w": 800,
  "h": 160,
  "z": 1,
  "opacity": 1,
  "rotation": 0,
  "locked": false,
  "hidden": false,
  "animation": { "in": "fade", "out": "fade", "duration_ms": 400 }
}
```

### Tipe elemen
- `text`: `font_family`, `font_size`, `font_weight`, `line_height`, `letter_spacing`, `color`, `align`, `vertical_align`, `bg`, `radius`, `padding`, `shadow`, `max_lines`, `auto_fit`.
- `image`: `url`, `fit`, `position`, `radius`, `fallback_url`.
- `video`: `url`, `fit`, `position`, `muted`, `loop`, `volume`, `autoplay`, `radius`, `fallback_url`.
- `website`: `url`, `fallback_url`, `reload_interval`, `zoom`, `radius` (online-only).
- `clock`: `timezone`, `format`, `locale`, `show_date`, `color`, `bg`, `align`, `font_size`.
- `ticker`: `text`, `direction`, `speed`, `gap`, `separator`, `font_size`, `color`, `bg`.
- `shape`: `shape_type` (`rect|circle`), `bg`, `border`, `radius`.
- `slot`: `slot` (`main`), `fit` (`cover|contain|fill`), `radius`; area tempat media playlist dirender.
- `qr`: `data`, `fg`, `bg`, `label`, `error_correction`.
- `countdown`: `target`, `timezone`, `format`, `label`, `color`, `bg`.

### Data binding
Teks/ticker mendukung token:
- `{{device.name}}`
- `{{device.location}}`
- `{{date}}`
- `{{time}}`
- `{{playlist.name}}`
- `{{hotel.name}}`

Client meresolve token saat render; server tidak mengirim nilai final.
Jika token tidak dikenal, tampilkan string kosong.

### Render
- Render **Layout Utama**: hitung skala dari `layout.width`/`layout.height` dan `fit`, gambar `letterbox` penuh layar, lalu gambar kanvas yang sudah diskalakan dan ditengahkan.
- Semua elemen dirender relatif terhadap kanvas utama, bukan langsung terhadap layar device.
- Saat memutar media biasa dan `device.main_layout_id` diset: render layout main, lalu render media ke slot (`fit` slot); elemen lain menjadi overlay. Tanpa slot/layout main → media full screen.
- Urutan render berdasarkan `z` asc, lalu id. `opacity` (0-1) berlaku per elemen; background gradient dirender sebagai CSS background biasa.
- `hidden=true` tidak dirender tetapi tetap disimpan.
- `locked=true` hanya memengaruhi editor admin, bukan render.
- Website hanya dirender jika online; jika tidak, gunakan `fallback_url`.
- Semua elemen wajib aman terhadap URL kosong (placeholder, bukan crash).

## 9. Monitoring & remote control

### WebSocket device
Pesan device → server:
```json
{ "type": "hello", "device_id": "screen-lobby-1", "app_version": "1.4.0" }
{ "type": "telemetry", "seq": 128, "ts": "...", "metrics": {} }
{ "type": "playback", "events": [] }
{ "type": "command_ack", "id": 91, "status": "ok", "result": {} }
{ "type": "error", "code": "asset_missing", "message": "..." }
```

Pesan server → device:
```json
{ "type": "welcome", "server_time": "...", "intervals": { "telemetry": 10, "heartbeat": 30 }, "commands": [] }
{ "type": "command", "id": 91, "action": "reload", "payload": {}, "expires_at": "..." }
{ "type": "config_changed", "version": "rev-13-..." }
{ "type": "ping", "ts": "..." }
```

Command:
- `sync_now`, `reload`, `clear_cache`, `screenshot`, `identify`,
- `restart`, `set_volume`, `set_brightness`, `set_orientation`, `set_resolution`.

Setiap command punya `id`, `expires_at`, status `pending|sent|acked|failed`, dan ack dari device.

### Telemetry
```json
{
  "type": "telemetry",
  "seq": 128,
  "ts": "2026-10-21T10:00:00Z",
  "app_version": "1.4.0",
  "uptime_seconds": 86400,
  "platform": { "client_type": "chromium", "os": "linux", "arch": "x64" },
  "network": { "type": "ethernet", "ip": "192.168.1.50", "latency_ms": 12, "rx_bytes": 0, "tx_bytes": 0 },
  "cpu": { "usage_percent": 18.5, "cores": 4, "temperature_c": null },
  "memory": { "total_bytes": null, "used_bytes": null, "available_bytes": null, "percent": null, "source": "unknown" },
  "disk": { "total_bytes": 128000000000, "used_bytes": 64000000000, "free_bytes": 64000000000, "percent": 50, "source": "browser-estimate" },
  "cache": { "manifest_version": "rev-12-abcd", "bytes": 850000000, "items": 42, "limit_bytes": 2147483648, "hit_rate": 0.98 },
  "display": { "width": 1920, "height": 1080, "orientation": "landscape", "volume": 80, "brightness": 80 },
  "playback": { "playlist_id": 3, "media_id": 17, "layout_id": 2, "position_seconds": 8, "duration_seconds": 15, "state": "playing" },
  "health": { "level": "ok", "last_error": null, "queue_depth": 0 }
}
```

Field hardware opsional. `source` wajib menjelaskan asal nilai:
- `os` — native/agent.
- `browser-estimate` — estimasi browser.
- `unknown` — tidak tersedia.

### Online/offline
- Client mengirim telemetry tiap 10 detik; state change dikirim langsung.
- Server menyimpan `last_seen_at`, `last_metrics_json`.
- Offline jika tidak ada telemetry/heartbeat selama 2× interval (default 20–30 detik).
- Server mencatat event `online`/`offline` dan mengirim ke admin WS.
- Jangan memakai kolom status admin (`pending/active/disabled`) untuk online/offline.

### Retensi
- `ds_device_metric_samples`: 1 menit, retensi 7 hari.
- `ds_device_metric_rollups`: 5 menit, retensi 90 hari.
- `ds_device_events`: 90 hari.
- `ds_commands`: 30–90 hari.
- Cleanup job periodik; bisa mengikuti pola cleanup log auth.

### Alert awal
- Offline > 2 interval.
- Disk free < 10%.
- RAM > 90%.
- Cache > 90% limit.
- Manifest stale > 24 jam.
- Playback error berulang (≥3 dalam 10 menit).
- Command gagal/expired.
- Token invalid.

## 10. Publish & rollback

- Data admin (media/layout/playlist/schedule/assignment) adalah **draft**.
- `POST /signage/publish` membangun snapshot immutable.
- Device sync hanya menerima snapshot published.
- Publish target: device, grup, atau semua.
- Preview: `POST /signage/publish/preview` memakai draft.
- Riwayat menyimpan `version`, `checksum`, `created_by`, `note`, `created_at`.
- Rollback membuat device menunjuk revision lama (revision tidak dihapus).
- Perubahan device settings (volume/brightness/timezone/cache) tidak perlu publish konten; berlaku pada sync berikutnya.

## 11. Migration

| Migration | Isi |
|---|---|
| `012_upgrade_signage_devices.sql` | Token hash/prefix, app version, IP, last sync, cache/storage metrics, current media/playlist, screenshot, timezone, cache limit, settings JSON, grup, approval. |
| `013_create_signage_publish.sql` | `ds_revisions`, `ds_publish_events`, `ds_commands`, grup device + member. |
| `014_upgrade_signage_media_meta.sql` | MIME, size, width/height, thumbnail, origin URL, checksum index. |
| `015_upgrade_signage_playlog.sql` | `event_id` unik, ended_at, error, item/layout reference. |
| `016_create_signage_monitoring.sql` | Metric samples, rollups, events, index retensi. |

Semua migration produksi dijalankan manual setelah backup DB `ttpg_digital_signage`.

## 12. Acceptance criteria

Status: backend/API dan smoke test lokal **DONE**; client/player di luar repo.

Backend/API:
- Token per device valid; token salah/revoked → 401/403.
- `/signage/sync` mengembalikan 304 saat ETag sama.
- Draft tidak terlihat device sebelum publish.
- Rollback mengembalikan manifest revision lama.
- Media download mendukung Range + checksum.
- Telemetry tersimpan dan online/offline terdeteksi.
- Command terkirim via WS dan fallback HTTP, ack tercatat.
- Retensi metrik berjalan.

Admin UI:
- Device detail menampilkan online, now playing, CPU/RAM/disk/cache, screenshot.
- Grafik 1 jam/24 jam/7 hari.
- Publish preview, history, rollback.
- Schedule CRUD berfungsi.
- Layout editor: media picker, undo/redo, snap, layer, QR, fallback website.

Client contract:
- Client bisa boot offline dari cache.
- Client memverifikasi checksum.
- Client menghitung jadwal offline.
- Client menampilkan fallback website saat offline.
- Client mengirim playback log idempoten setelah reconnect.

## 13. Keamanan

- Token device tidak boleh muncul di log, URL publik, atau respons list.
- WS ticket sekali pakai; admin WS wajib JWT role admin.
- Batasi ukuran payload telemetry (mis. 32KB) dan command payload.
- Validasi semua angka/string; jangan simpan HTML mentah dari client.
- Rate limit endpoint runtime.
- Untuk konten sensitif, pertimbangkan signed URL media. Saat ini media `/uploads/signage` bersifat publik.
- Gunakan `wss://` di produksi.

## 14. Catatan & batasan

- Browser tidak bisa membaca RAM/HDD total secara akurat; gunakan `navigator.deviceMemory`, `navigator.hardwareConcurrency`, dan `StorageManager.estimate()` sebagai estimasi.
- Untuk metrik akurat, client perlu Electron/native agent yang membaca OS.
- Website cross-origin tidak bisa diandalkan offline; fallback wajib.
- Hub WebSocket saat ini direncanakan in-memory (satu instance server). Jika nanti multi-instance, perlu Redis/NATS pub-sub.
- Retensi 7 hari raw + 90 hari rollup perlu dipantau jika jumlah device sangat besar.
