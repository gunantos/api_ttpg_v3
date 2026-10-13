package handler

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/gin-gonic/gin"
)

type printBoardingpassData struct {
	Noreg            string
	TicketID         string
	NamaPNP          string
	NamaPO           string
	Tujuan           string
	TanggalBerangkat string
	JamBerangkat     string
	PrintedAt        string
}

func PrintBoardingpass(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "kode print wajib diisi"})
		return
	}

	data := getPrintBoardingpassDataFromQuery(c, code)
	if data == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var err error
		data, err = getPrintBoardingpassData(ctx, code)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": "boardingpass tidak ditemukan"})
			return
		}
		if err != nil {
			logger.Error("get print boardingpass failed code=%s err=%v", code, err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal mengambil data boardingpass"})
			return
		}
	}

	renderPrintBoardingpass(c, data)
}

func getPrintBoardingpassDataFromQuery(c *gin.Context, code string) *printBoardingpassData {
	namaPNP := strings.TrimSpace(c.Query("nama_pnp"))
	namaPO := strings.TrimSpace(c.Query("nama_po"))
	tujuan := strings.TrimSpace(c.Query("tujuan"))
	tanggalBerangkat := strings.TrimSpace(c.Query("tanggal_berangkat"))
	jamBerangkat := strings.TrimSpace(c.Query("jam_berangkat"))

	if namaPNP == "" || namaPO == "" || tujuan == "" || tanggalBerangkat == "" || jamBerangkat == "" {
		return nil
	}

	return &printBoardingpassData{
		Noreg:            code,
		TicketID:         strings.TrimSpace(c.Query("ticket_id")),
		NamaPNP:          namaPNP,
		NamaPO:           namaPO,
		Tujuan:           tujuan,
		TanggalBerangkat: tanggalBerangkat,
		JamBerangkat:     jamBerangkat,
		PrintedAt:        formatIndonesianDateTime(time.Now()),
	}
}

func renderPrintBoardingpass(c *gin.Context, data *printBoardingpassData) {
	var html bytes.Buffer
	if err := boardingpassPrintTemplate.Execute(&html, data); err != nil {
		logger.Error("render print boardingpass failed noreg=%s err=%v", data.Noreg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal render boardingpass"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html.String())
}

func getPrintBoardingpassData(ctx context.Context, code string) (*printBoardingpassData, error) {
	var data printBoardingpassData
	var tujuan sql.NullString

	err := db.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE(bp.noreg, ''),
			COALESCE(bp.ticket_id, ''),
			COALESCE(bp.nama_pnp, ''),
			CASE
				WHEN bp.id_po = 'PO028' THEN 'Damri'
				ELSE COALESCE(NULLIF(bp.id_po, ''), '-')
			END,
			t.tujuan,
			DATE_FORMAT(bp.tanggal_berangkat, '%Y-%m-%d'),
			TIME_FORMAT(bp.jam_berangkat, '%H:%i')
		FROM terminal_boardingpass bp
		LEFT JOIN tbl_tujuan t ON t.id_tujuan = bp.id_tujuan
		WHERE (bp.noreg = ? OR bp.ticket_id = ?)
		AND bp.deleted IS NULL
		ORDER BY bp.id DESC
		LIMIT 1
	`, code, code).Scan(
		&data.Noreg,
		&data.TicketID,
		&data.NamaPNP,
		&data.NamaPO,
		&tujuan,
		&data.TanggalBerangkat,
		&data.JamBerangkat,
	)
	if err != nil {
		return nil, err
	}

	if tujuan.Valid && strings.TrimSpace(tujuan.String) != "" {
		data.Tujuan = tujuan.String
	} else {
		data.Tujuan = "-"
	}
	data.PrintedAt = formatIndonesianDateTime(time.Now())

	return &data, nil
}

func formatIndonesianDateTime(t time.Time) string {
	months := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	return t.Format("2 ") + months[int(t.Month())-1] + t.Format(" 2006 (15:04:05)")
}

var boardingpassPrintTemplate = template.Must(template.New("boardingpass-print").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Print Boardingpass {{.Noreg}}</title>
  <style>
    @page { margin: 0; page-break-before: always; }
    body {
      font-size: 12px;
      width: 78mm;
      padding-top: 0;
      margin-top: 0;
      padding: 10px;
      padding-bottom: 20px;
    }
    table {
      width: 100%;
      display: table;
      border-collapse: collapse;
      border-spacing: 0;
      font-family: monospace, cursive;
      line-height: 16px;
      text-align: left;
    }
    .header {
      font-size: 14px;
      font-weight: 800;
      line-height: 5px;
      text-align: center;
      margin-bottom: 20px;
    }
    #qrcode {
      text-align: center;
      display: flex;
      justify-content: center;
      margin-top: 10px;
      min-height: 60px;
    }
  </style>
</head>
<body>
<section id="main">
  <div class="header">
    <p>BOARDINGPASS</p>
    <p>TERMINAL TERPADU PULOGEBANG</p>
    <p style="font-size: 10px;border-bottom: 1px double #000;padding-bottom: 10px;">{{.PrintedAt}}</p>
    <div id="qrcode"></div>
    <table width="100%" style="margin-top: 20px;">
      <tbody>
        <tr>
          <td width="40%">Nama</td>
          <td>: <span>{{.NamaPNP}}</span></td>
        </tr>
        <tr style="display: none;">
          <td>Tiket ID</td>
          <td>: <span>{{.TicketID}}</span></td>
        </tr>
        <tr>
          <td>Nama PO</td>
          <td>: <span>{{.NamaPO}}</span></td>
        </tr>
        <tr>
          <td>Tujuan</td>
          <td>: <span>{{.Tujuan}}</span></td>
        </tr>
        <tr>
          <td>Tanggal</td>
          <td>: <span>{{.TanggalBerangkat}}</span></td>
        </tr>
        <tr>
          <td>Jam</td>
          <td>: <span>{{.JamBerangkat}}</span></td>
        </tr>
      </tbody>
    </table>
    <p></p>
    <p>==###==</p>
  </div>
</section>
<script src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
<script>
  const noreg = {{printf "%q" .Noreg}};
  function renderFallback() {
    document.getElementById("qrcode").innerText = noreg;
  }
  if (window.QRCode) {
    new QRCode("qrcode", {
      text: noreg,
      width: 60,
      height: 60,
      colorDark: "#000000",
      colorLight: "#ffffff",
      correctLevel: QRCode.CorrectLevel.H
    });
  } else {
    renderFallback();
  }
  window.addEventListener("load", function () {
    const autoprint = new URLSearchParams(window.location.search).get("autoprint");
    if (autoprint === "1") {
      setTimeout(function () { window.print(); }, 300);
    }
  });
</script>
</body>
</html>`))
