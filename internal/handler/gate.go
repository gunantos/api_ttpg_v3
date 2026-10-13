package handler

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/client"
	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/model"
	"github.com/andtho89/ttpg/internal/request"
	"github.com/andtho89/ttpg/internal/response"
	"github.com/gin-gonic/gin"
)

func getIDTujuanFromTujuanTx(tx *sql.Tx, tujuan string) (*int, error) {
	if tujuan == "" {
		return nil, nil
	}

	var id int

	err := tx.QueryRow(`
		SELECT id_tujuan
		FROM tbl_tujuan
		WHERE LOWER(TRIM(tujuan)) = LOWER(?)
		LIMIT 1
	`, tujuan).Scan(&id)

	if err == nil {
		return &id, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	result, err := tx.Exec(`
		INSERT INTO tbl_tujuan (
			provinsi,
			tujuan,
			kawasan
		)
		VALUES (?, ?, ?)
	`, "NN", tujuan, "NN")

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			err = tx.QueryRow(`
				SELECT id_tujuan
				FROM tbl_tujuan
				WHERE tujuan = ?
				LIMIT 1
			`, tujuan).Scan(&id)

			if err != nil {
				return nil, err
			}

			return &id, nil
		}

		return nil, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	newID := int(lastID)

	return &newID, nil
}

func scanBoardingpass(row *sql.Row) (*model.Boardingpass, error) {
	var b model.Boardingpass

	err := row.Scan(
		&b.ID,
		&b.Noreg,
		&b.TicketID,
		&b.IDPO,
		&b.IDTujuan,
		&b.Nopol,
		&b.TanggalBerangkat,
		&b.JamBerangkat,
		&b.NamaPnp,
		&b.SexPnp,
		&b.Fungsi,
		&b.GateID,
		&b.DeviceID,
		&b.TimeTab,
		&b.IDUser,
		&b.Retribusi,
		&b.CountScan,
		&b.PembelianDari,
		&b.Created,
		&b.Updated,
		&b.Deleted,
	)

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func Gate(c *gin.Context) {
	var req request.GateRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body tidak valid"})
		return
	}

	if req.Noreg == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "noreg wajib diisi"})
		return
	}

	if config.C.Bypass {
		c.JSON(http.StatusOK, response.GateResponse{Success: true, Reason: "BYPASS MODE"})
		return
	}

	noreg := req.Noreg

	deviceID := c.GetHeader("X-Device-ID")
	if deviceID == "" {
		deviceID = c.ClientIP()
	}
	logger.Info("GATE REQUEST noreg=%s gate_id=%d device=%s", noreg, req.GateID, deviceID)

	if config.C.IsIDCardAllowed(noreg) {
		c.JSON(http.StatusOK, response.GateResponse{Success: true, Reason: "VALID via LIST_ID_CARD"})
		return
	}

	row := db.DB.QueryRow(`
		SELECT id, noreg, ticket_id, id_po, id_tujuan, nopol,
			tanggal_berangkat, jam_berangkat, nama_pnp, sex_pnp,
			fungsi, gate_id, device_id, time_tab, id_user,
			retribusi, count_scan, pembelian_dari, created, updated, deleted
		FROM terminal_boardingpass
		WHERE fungsi IN ('checking','checkin','gatein','gateout')
		AND noreg = ?
		AND id_po IS NOT NULL
		AND TRIM(id_po) != ''
		AND DATE(tanggal_berangkat) = CURDATE()
		ORDER BY id DESC
		LIMIT 1
	`, noreg)

	b, err := scanBoardingpass(row)

	if err == sql.ErrNoRows {
		ticketClient := client.NewTicketClient()
		logger.Info("CHECK API EXTERNAL noreg=%s", noreg)
		ticketData, err := ticketClient.GetBarcodeData(noreg)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logger.Error("timeout calling ticket system noreg=%s err=%v", noreg, err)
				c.JSON(http.StatusOK, response.GateResponse{Success: false, Reason: "TIMEOUT SISTEM TIKET"})
				return
			}
			var netErr net.Error
			if errors.As(err, &netErr) {
				logger.Error("connection to ticket system failed noreg=%s err=%v", noreg, err)
				c.JSON(http.StatusOK, response.GateResponse{Success: false, Reason: "KONEKSI KE SISTEM TIKET GAGAL"})
				return
			}
			logger.Error("get barcode data failed noreg=%s err=%v", noreg, err)
			c.JSON(http.StatusOK, response.GateResponse{Success: false, Reason: "BOARDINGPASS TIDAK DITEMUKAN"})
			return
		}

		tx, err := db.DB.Begin()
		if err != nil {
			logger.Error("begin transaction failed noreg=%s err=%v", noreg, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaction"})
			return
		}
		defer tx.Rollback()
		if ticketData.NoPolisi == "" {
			logger.Error("no polisi is empty noreg=%s", noreg)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no polisi is empty"})
			return
		}
		idTujuan, err := getIDTujuanFromTujuanTx(tx, ticketData.Turun)
		if err != nil {
			logger.Error("get id tujuan failed noreg=%s err=%v", noreg, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal get tujuan"})
			return
		}

		result, err := tx.Exec(`
			INSERT INTO terminal_boardingpass (
				noreg,
				ticket_id,
				id_po,
				id_tujuan,
				nopol,
				tanggal_berangkat,
				jam_berangkat,
				nama_pnp,
				sex_pnp,
				fungsi,
				retribusi,
				count_scan,
				pembelian_dari,
				gate_id,
				device_id,
				time_tab,
				created,
				updated
			)
			VALUES (
				?,
				?,
				?,
				?,
				?,
				?,
				?,
				?,
				?,
				'gatein',
				0,
				1,
				?,
				?,
				?,
				NOW(),
				NOW(),
				NOW()
			)
		`,
			ticketData.NoBarcode,
			ticketData.NoBarcode,
			"PO122",
			idTujuan,
			ticketData.NoPolisi,
			ticketData.TanggalBerangkat,
			ticketData.JamBerangkat,
			ticketData.NamaPenumpang,
			nil,
			"Loket Terminal",
			req.GateID,
			deviceID,
		)

		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				row := db.DB.QueryRow(`
					SELECT id, noreg, ticket_id, id_po, id_tujuan, nopol,
						   tanggal_berangkat, jam_berangkat, nama_pnp, sex_pnp,
						   fungsi, gate_id, device_id, time_tab, id_user,
						   retribusi, count_scan, pembelian_dari, created, updated, deleted
					FROM terminal_boardingpass
					WHERE noreg = ?
					ORDER BY id DESC
					LIMIT 1
				`, noreg)

				b, err = scanBoardingpass(row)
				if err != nil {
					logger.Error("scan boardingpass failed noreg=%s err=%v", noreg, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal ambil duplicate boardingpass"})
					return
				}
			} else {
				logger.Error("insert boardingpass failed noreg=%s err=%v", noreg, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal insert boardingpass"})
				return
			}
		} else {
			lastID, _ := result.LastInsertId()

			row := tx.QueryRow(`
				SELECT id, noreg, ticket_id, id_po, id_tujuan, nopol,
					   tanggal_berangkat, jam_berangkat, nama_pnp, sex_pnp,
					   fungsi, gate_id, device_id, time_tab, id_user,
					   retribusi, count_scan, pembelian_dari, created, updated, deleted
				FROM terminal_boardingpass
				WHERE id = ?
			`, lastID)

			b, err = scanBoardingpass(row)
			if err != nil {
				logger.Error("scan boardingpass failed noreg=%s err=%v", noreg, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal ambil boardingpass baru"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			logger.Error("commit transaction failed noreg=%s err=%v", noreg, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal commit transaction"})
			return
		}

		logger.Info("GATE API INSERT noreg=%s", noreg)

		c.JSON(http.StatusOK, response.GateResponse{Success: true, Reason: "VALID via API TICKET"})
		return
	}

	if err != nil {
		logger.Error("check boardingpass failed noreg=%s err=%v", noreg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fungsi := ""
	if b.Fungsi.Valid {
		fungsi = b.Fungsi.String
	}

	countScan := 0
	if b.CountScan.Valid {
		countScan = int(b.CountScan.Int64)
	}

	allowed := false
	reason := ""

	switch fungsi {
	case "checking", "checkin", "":
		allowed = true
		reason = "VALID CHECKIN"

	case "gatein", "gateout":
		if countScan >= 5 {
			allowed = false
			reason = "MAXIMUM SCAN TERCAPAI"
			break
		}
		if b.TimeTab.Valid {
			diff := time.Since(b.TimeTab.Time)
			if diff > 5*time.Minute {
				allowed = false
				reason = "SCAN SUDAH EXPIRED"
				break
			}
		}
		allowed = true
		reason = "VALID RESCAN"

	default:
		allowed = false
		reason = "STATUS BOARDINGPASS TIDAK VALID"
	}

	if !allowed {
		logger.Warning("GATE DENIED noreg=%s reason=%s", noreg, reason)
		c.JSON(http.StatusOK, response.GateResponse{Success: false, Reason: reason})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE terminal_boardingpass
		SET
			fungsi = 'gatein',
			count_scan = COALESCE(count_scan,0) + 1,
			time_tab = NOW(),
			gate_id = ?,
			device_id = ?,
			updated = NOW()
		WHERE id = ?
	`,
		req.GateID,
		deviceID,
		b.ID,
	)

	if err != nil {
		logger.Error("update gate failed noreg=%s err=%v", noreg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal update gate"})
		return
	}
	logger.Info("GATE SUCCESS noreg=%s reason=%s", noreg, reason)
	c.JSON(http.StatusOK, response.GateResponse{Success: true, Reason: reason})
}
