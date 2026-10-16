package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/config"

	"github.com/andtho89/ttpg/internal/client"
	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/model"
	"github.com/andtho89/ttpg/internal/request"
	"github.com/andtho89/ttpg/internal/response"
	"github.com/gin-gonic/gin"
)

func Create(c *gin.Context) {
	var req request.CreateBoardingpassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body tidak valid: " + err.Error()})
		return
	}

	ticketClient := client.NewTicketClient()
	ticketData, err := ticketClient.GetBarcodeData(req.Noreg)
	if err != nil {
		log.Printf("[ERROR] API check failed for noreg %s: %v\n", req.Noreg, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "noreg tidak ditemukan di sistem tiket"})
		return
	}

	namaPnp := req.NamaPnp
	if namaPnp == "" {
		namaPnp = ticketData.NamaPenumpang
	}

	tanggalBerangkatStr := req.TanggalBerangkat
	if tanggalBerangkatStr == "" {
		tanggalBerangkatStr = ticketData.TanggalBerangkat
	}

	jamBerangkat := req.JamBerangkat
	if jamBerangkat == "" {
		jamBerangkat = ticketData.JamBerangkat
	}

	nabbaspol := req.Nopol
	if nabbaspol == "" {
		nabbaspol = ticketData.NoPolisi
	}
	nopol := nabbaspol

	var existingCount int
	checkQuery := "SELECT COUNT(*) FROM terminal_boardingpass WHERE noreg = ? AND deleted IS NULL"
	err = db.DB.QueryRow(checkQuery, req.Noreg).Scan(&existingCount)
	if err != nil {
		log.Printf("[ERROR] Check existing boardingpass: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memeriksa boardingpass yang ada"})
		return
	}

	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "boardingpass dengan noreg ini sudah ada"})
		return
	}

	const query = `
        INSERT INTO terminal_boardingpass
        (noreg, ticket_id, id_po, id_tujuan, nopol, tanggal_berangkat, jam_berangkat,
         nama_pnp, sex_pnp, fungsi, retribusi, pembelian_dari, created)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
    `

	result, err := db.DB.Exec(query,
		req.Noreg, req.TicketID, req.IDPO, req.IDTujuan, nopol,
		tanggalBerangkatStr, jamBerangkat,
		namaPnp, req.SexPnp, req.Fungsi, req.Retribusi, req.PembelianDari,
	)
	if err != nil {
		log.Printf("[ERROR] Create boardingpass: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal insert boardingpass"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"message":  "boardingpass berhasil dibuat",
		"api_data": ticketData,
	})
}

func GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	b, err := selectBoardingpass(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "boardingpass tidak ditemukan"})
		return
	}
	if err != nil {
		log.Printf("[ERROR] GetByID: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal query boardingpass"})
		return
	}

	c.JSON(http.StatusOK, b)
}

func selectBoardingpass(id int) (model.Boardingpass, error) {
	const query = `
        SELECT id, noreg, ticket_id, id_po, id_tujuan, nopol,
               tanggal_berangkat, jam_berangkat, nama_pnp, sex_pnp,
               fungsi, gate_id, device_id, time_tab, id_user,
               retribusi, count_scan, pembelian_dari, created, updated, deleted,
               ktp_photo, ticket_photo
        FROM terminal_boardingpass
        WHERE id = ? AND deleted IS NULL
    `

	var b model.Boardingpass
	err := db.DB.QueryRow(query, id).Scan(
		&b.ID, &b.Noreg, &b.TicketID, &b.IDPO, &b.IDTujuan,
		&b.Nopol, &b.TanggalBerangkat, &b.JamBerangkat,
		&b.NamaPnp, &b.SexPnp, &b.Fungsi, &b.GateID,
		&b.DeviceID, &b.TimeTab, &b.IDUser, &b.Retribusi,
		&b.CountScan, &b.PembelianDari, &b.Created, &b.Updated, &b.Deleted,
		&b.KTPPhoto, &b.TicketPhoto,
	)
	return b, err
}

func List(c *gin.Context) {
	where := "deleted IS NULL"
	args := []interface{}{}

	if v := strings.TrimSpace(c.Query("noreg")); v != "" {
		where += " AND noreg LIKE ?"
		args = append(args, "%"+v+"%")
	}
	if v := strings.TrimSpace(c.Query("id_po")); v != "" {
		where += " AND id_po = ?"
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("fungsi")); v != "" {
		where += " AND fungsi = ?"
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("id_tujuan")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id_tujuan tidak valid"})
			return
		}
		where += " AND id_tujuan = ?"
		args = append(args, n)
	}

	limit := 100
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit harus 1-1000"})
			return
		}
		limit = n
	}
	offset := 0
	if v := strings.TrimSpace(c.Query("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset tidak valid"})
			return
		}
		offset = n
	}

	var total int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM terminal_boardingpass WHERE "+where, args...).Scan(&total); err != nil {
		log.Printf("[ERROR] List count: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menghitung boardingpass"})
		return
	}

	dataQuery := `
        SELECT id, noreg, id_po, id_tujuan, nopol, tanggal_berangkat,
               jam_berangkat, nama_pnp, fungsi, count_scan,
               DATE_FORMAT(created, '%Y-%m-%d %H:%i:%s')
        FROM terminal_boardingpass
        WHERE ` + where + `
        ORDER BY id DESC
        LIMIT ? OFFSET ?`

	rows, err := db.DB.Query(dataQuery, append(args, limit, offset)...)
	if err != nil {
		log.Printf("[ERROR] List query: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal query boardingpass"})
		return
	}
	defer rows.Close()

	list := response.ListBoardingpassResponse{Data: []response.BoardingpassResponse{}, Total: total}
	for rows.Next() {
		var r response.BoardingpassResponse
		var idTujuan, countScan sql.NullInt64
		var fungsi sql.NullString
		if err := rows.Scan(&r.ID, &r.Noreg, &r.IDPO, &idTujuan, &r.Nopol,
			&r.TanggalBerangkat, &r.JamBerangkat, &r.NamaPnp, &fungsi,
			&countScan, &r.Created); err != nil {
			log.Printf("[ERROR] List scan: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal decode boardingpass"})
			return
		}
		r.IDTujuan = int(idTujuan.Int64)
		r.Fungsi = fungsi.String
		r.CountScan = int(countScan.Int64)
		list.Data = append(list.Data, r)
	}
	c.JSON(http.StatusOK, list)
}

func Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	var req request.UpdateBoardingpassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body tidak valid: " + err.Error()})
		return
	}

	sets := []string{}
	args := []interface{}{}

	if req.Fungsi != nil {
		sets = append(sets, "fungsi = ?")
		args = append(args, strings.TrimSpace(*req.Fungsi))
	}
	if req.GateID != nil {
		v := strings.TrimSpace(*req.GateID)
		if v == "" {
			sets = append(sets, "gate_id = NULL")
		} else {
			n, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "gate_id tidak valid"})
				return
			}
			sets = append(sets, "gate_id = ?")
			args = append(args, n)
		}
	}
	if req.DeviceID != nil {
		sets = append(sets, "device_id = ?")
		args = append(args, strings.TrimSpace(*req.DeviceID))
	}
	if req.TimeTab != nil {
		v := strings.TrimSpace(*req.TimeTab)
		if v == "" {
			sets = append(sets, "time_tab = NULL")
		} else {
			var parsed time.Time
			var perr error
			for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", time.RFC3339, "2006-01-02 15:04"} {
				if parsed, perr = time.Parse(layout, v); perr == nil {
					break
				}
			}
			if perr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "time_tab tidak valid (format YYYY-MM-DD HH:MM:SS)"})
				return
			}
			sets = append(sets, "time_tab = ?")
			args = append(args, parsed.Format("2006-01-02 15:04:05"))
		}
	}
	if req.CountScan != nil {
		if *req.CountScan < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "count_scan tidak boleh negatif"})
			return
		}
		sets = append(sets, "count_scan = ?")
		args = append(args, *req.CountScan)
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada field yang diupdate"})
		return
	}
	sets = append(sets, "updated = NOW()")

	query := "UPDATE terminal_boardingpass SET " + strings.Join(sets, ", ") + " WHERE id = ? AND deleted IS NULL"
	result, err := db.DB.Exec(query, append(args, id)...)
	if err != nil {
		log.Printf("[ERROR] Update boardingpass: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal update boardingpass"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "boardingpass tidak ditemukan"})
		return
	}

	b, err := selectBoardingpass(id)
	if err != nil {
		log.Printf("[ERROR] Update read-back: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membaca boardingpass"})
		return
	}
	c.JSON(http.StatusOK, b)
}

func Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}

	result, err := db.DB.Exec(
		"UPDATE terminal_boardingpass SET deleted = NOW(), updated = NOW() WHERE id = ? AND deleted IS NULL",
		id,
	)
	if err != nil {
		log.Printf("[ERROR] Delete boardingpass: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal hapus boardingpass"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "boardingpass tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "boardingpass berhasil dihapus"})
}

const bpImgMaxBytes = 5 << 20

func bpImageDir(kind string) string {
	return filepath.Join(config.C.UploadDir, "boardingpass", kind)
}

func bpImageName(id int, kind, ext string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("bp%d_%s_%d_%s.%s", id, kind, time.Now().Unix(), hex.EncodeToString(b), ext)
}

func bpKindColumn(kind string) (string, bool) {
	switch kind {
	case "ktp":
		return "ktp_photo", true
	case "tiket":
		return "ticket_photo", true
	}
	return "", false
}

func bpLoad(c *gin.Context, id int) bool {
	var n int
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM terminal_boardingpass WHERE id=? AND deleted IS NULL", id).Scan(&n); err != nil || n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "boardingpass tidak ditemukan"})
		return false
	}
	return true
}

func bpKindFromPath(c *gin.Context) string {
	p := strings.TrimRight(c.Request.URL.Path, "/")
	if strings.HasSuffix(p, "/ktp") {
		return "ktp"
	}
	if strings.HasSuffix(p, "/tiket") {
		return "tiket"
	}
	if v := c.Param("kind"); v != "" {
		return v
	}
	return ""
}

func UploadBoardingpassImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	kind := bpKindFromPath(c)
	col, ok := bpKindColumn(kind)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis gambar tidak valid (ktp|tiket)"})
		return
	}
	if !bpLoad(c, id) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bpImgMaxBytes+1024)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file wajib dikirim (multipart form-field 'file')"})
		return
	}
	if file.Size > bpImgMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ukuran gambar maksimal 5MB"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membaca file: " + err.Error()})
		return
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	var ext string
	switch http.DetectContentType(head[:n]) {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format gambar harus JPG, PNG, atau WebP"})
		return
	}
	dir := bpImageDir(kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat folder upload: " + err.Error()})
		return
	}
	filename := bpImageName(id, kind, ext)
	if err := c.SaveUploadedFile(file, filepath.Join(dir, filename)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan file: " + err.Error()})
		return
	}
	// hapus file lama bila ada
	var old sql.NullString
	var colname string
	switch kind {
	case "ktp":
		colname = "ktp_photo"
	case "tiket":
		colname = "ticket_photo"
	}
	_ = colname
	qr := "SELECT " + col + " FROM terminal_boardingpass WHERE id=? AND deleted IS NULL"
	if err := db.DB.QueryRow(qr, id).Scan(&old); err == nil && old.Valid && old.String != "" {
		os.Remove(filepath.Join(dir, filepath.Base(old.String)))
	}
	if _, err := db.DB.Exec("UPDATE terminal_boardingpass SET "+col+"=?, updated=NOW() WHERE id=? AND deleted IS NULL", filename, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan data gambar: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "kind": kind, col: filename, "message": "gambar berhasil disimpan"})
}

func GetBoardingpassImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	kind := bpKindFromPath(c)
	col, ok := bpKindColumn(kind)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis gambar tidak valid (ktp|tiket)"})
		return
	}
	if !bpLoad(c, id) {
		return
	}
	var name sql.NullString
	qr := "SELECT " + col + " FROM terminal_boardingpass WHERE id=? AND deleted IS NULL"
	if err := db.DB.QueryRow(qr, id).Scan(&name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membaca data boardingpass"})
		return
	}
	if !name.Valid || name.String == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "gambar belum ada"})
		return
	}
	fname := filepath.Base(name.String)
	if fname != name.String {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nama file gambar tidak valid"})
		return
	}
	path := filepath.Join(bpImageDir(kind), fname)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file gambar tidak ditemukan"})
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.File(path)
}

func DeleteBoardingpassImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	kind := bpKindFromPath(c)
	col, ok := bpKindColumn(kind)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis gambar tidak valid (ktp|tiket)"})
		return
	}
	if !bpLoad(c, id) {
		return
	}
	var name sql.NullString
	qr := "SELECT " + col + " FROM terminal_boardingpass WHERE id=? AND deleted IS NULL"
	if err := db.DB.QueryRow(qr, id).Scan(&name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membaca data boardingpass"})
		return
	}
	if name.Valid && name.String != "" {
		os.Remove(filepath.Join(bpImageDir(kind), filepath.Base(name.String)))
	}
	if _, err := db.DB.Exec("UPDATE terminal_boardingpass SET "+col+"=NULL, updated=NOW() WHERE id=? AND deleted IS NULL", id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menghapus data gambar"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "kind": kind, col: nil, "message": "gambar berhasil dihapus"})
}
