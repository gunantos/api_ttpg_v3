package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type penginapanReceiptRow struct {
	Nama   string
	Kamar  string
	Jumlah string
}

type penginapanReceiptData struct {
	Hotel     string
	PaymentID string
	Tanggal   string
	Status    string
	IsVoid    bool
	Metode    string
	Sumber    string
	Catatan   string
	Dibayar   string
	Rows      []penginapanReceiptRow
	Total     string
	Autoprint bool
}

func PaymentReceipt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
		return
	}

	var p models.Payment
	if err := db.PenginapanDB.
		Preload("PaymentMethod").
		Preload("Source").
		Preload("Guests.Guest.Room").
		First(&p, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}

	data := penginapanReceiptData{
		Hotel:     "PENGINAPAN",
		PaymentID: "#" + strconv.Itoa(p.ID),
		Tanggal:   formatIndonesianDateTime(time.Now()),
		Status:    strings.ToUpper(p.Status),
		Metode:    "-",
		Sumber:    "-",
		Catatan:   "-",
		Dibayar:   "-",
		Total:     formatRupiah(p.Amount),
		Autoprint: c.Query("autoprint") == "1",
	}

	var cfg models.PenginapanConfig
	if err := db.PenginapanDB.Where("key_configs = ?", "nama_hotel").First(&cfg).Error; err == nil &&
		cfg.ValueConfigs != nil && strings.TrimSpace(*cfg.ValueConfigs) != "" {
		data.Hotel = strings.TrimSpace(*cfg.ValueConfigs)
	}

	switch p.Status {
	case "lunas":
		data.Status = "LUNAS"
	case "pending":
		data.Status = "MENUNGGU PEMBAYARAN"
	case "batal":
		data.Status = "DIBATALKAN"
		data.IsVoid = true
	default:
		data.Status = strings.ToUpper(p.Status)
	}

	if p.PaymentMethod != nil {
		data.Metode = p.PaymentMethod.Name
		if code := strings.TrimSpace(p.PaymentMethod.Code); code != "" {
			data.Metode = fmt.Sprintf("%s (%s)", p.PaymentMethod.Name, code)
		}
	}
	if p.Source != nil {
		data.Sumber = p.Source.Name
	}
	if p.Note != nil && strings.TrimSpace(*p.Note) != "" {
		data.Catatan = strings.TrimSpace(*p.Note)
	}
	if p.PaidAt != nil {
		data.Dibayar = formatIndonesianDateTime(*p.PaidAt)
	}

	for _, pg := range p.Guests {
		nama := "-"
		kamar := "-"
		if pg.Guest != nil {
			nama = pg.Guest.Name
			if pg.Guest.Room != nil {
				kamar = pg.Guest.Room.RoomNumber
			}
		}
		data.Rows = append(data.Rows, penginapanReceiptRow{
			Nama:   nama,
			Kamar:  kamar,
			Jumlah: formatRupiah(pg.Amount),
		})
	}

	var buf bytes.Buffer
	if err := penginapanReceiptTemplate.Execute(&buf, data); err != nil {
		logger.Error("render struk pembayaran failed payment_id=%d err=%v", p.ID, err)
		response.Error(c, http.StatusInternalServerError, "Gagal render struk", nil)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func formatRupiah(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}

var penginapanReceiptTemplate = template.Must(template.New("penginapan-receipt").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Struk Pembayaran {{.PaymentID}}</title>
  <style>
    @page { margin: 0; }
    body {
      font-size: 12px;
      width: 78mm;
      margin: 0;
      padding: 10px;
      font-family: monospace, cursive;
      line-height: 16px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      text-align: left;
    }
    .header {
      font-size: 14px;
      font-weight: 800;
      text-align: center;
      line-height: 16px;
      margin-bottom: 8px;
    }
    .title { font-size: 13px; font-weight: 800; text-align: center; }
    .void {
      text-align: center;
      font-size: 16px;
      font-weight: 800;
      letter-spacing: 2px;
      border: 2px solid #000;
      padding: 2px;
      margin: 6px 0;
    }
    .sep { border-bottom: 1px dashed #000; margin: 6px 0; }
    .total-row td { font-weight: 800; font-size: 13px; border-top: 1px solid #000; padding-top: 4px; }
    .guests td, .guests th { padding: 1px 0; vertical-align: top; }
    .guests th { text-align: left; font-weight: 800; }
    .guests .num { text-align: right; }
    .footer { text-align: center; margin-top: 10px; font-size: 11px; }
  </style>
</head>
<body>
  <div class="header">
    {{.Hotel}}<br>
    <span class="title">STRUK PEMBAYARAN</span><br>
    {{.Tanggal}}
  </div>
  {{if .IsVoid}}<div class="void">DIBATALKAN</div>{{end}}
  <div class="sep"></div>
  <table>
    <tbody>
      <tr><td>No.</td><td>: {{.PaymentID}}</td></tr>
      <tr><td>Status</td><td>: {{.Status}}</td></tr>
      <tr><td>Metode</td><td>: {{.Metode}}</td></tr>
      <tr><td>Sumber</td><td>: {{.Sumber}}</td></tr>
      <tr><td>Dibayar</td><td>: {{.Dibayar}}</td></tr>
      {{if ne .Catatan "-"}}<tr><td>Catatan</td><td>: {{.Catatan}}</td></tr>{{end}}
    </tbody>
  </table>
  <div class="sep"></div>
  <table class="guests">
    <thead>
      <tr><th>Tamu</th><th>Kamar</th><th class="num">Jumlah</th></tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr><td>{{.Nama}}</td><td>{{.Kamar}}</td><td class="num">{{.Jumlah}}</td></tr>
      {{else}}
      <tr><td colspan="3">-</td></tr>
      {{end}}
      <tr class="total-row"><td colspan="2">TOTAL</td><td class="num">{{.Total}}</td></tr>
    </tbody>
  </table>
  <div class="sep"></div>
  <div class="footer">Terima kasih atas kunjungan Anda</div>
  {{if .Autoprint}}
  <script>
    window.addEventListener("load", function () {
      setTimeout(function () { window.print(); }, 300);
    });
  </script>
  {{end}}
</body>
</html>`))
