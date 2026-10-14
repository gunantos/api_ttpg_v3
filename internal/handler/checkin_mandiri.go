package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andtho89/ttpg/internal/client"
	"github.com/andtho89/ttpg/internal/config"
	"github.com/andtho89/ttpg/internal/db"
	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/request"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	damriClientMu     sync.Mutex
	damriClientCached *client.DamriClient
)

type mongoBoarding struct {
	BoardingID      int64          `bson:"boarding_id"`
	TicketID        string         `bson:"ticket_id"`
	DateOfDeparture string         `bson:"date_of_departure"`
	TimeOfDeparture string         `bson:"time_of_departure"`
	Nopol           string         `bson:"nopol"`
	POID            string         `bson:"po_id"`
	DestinationID   models.FlexInt `bson:"destination_id"`
	Destination     struct {
		Tujuan string `bson:"tujuan"`
	} `bson:"destination"`
	Status string `bson:"status"`
	PO     struct {
		NamaPO      string `bson:"nama_po"`
		NamaSingkat string `bson:"nama_singkat"`
	} `bson:"po"`
	UserID *int `bson:"user_id"`
	User   struct {
		Username string `bson:"username"`
	} `bson:"user"`
	PNP []struct {
		Name   string  `bson:"name"`
		Gender *string `bson:"gender"`
		Status *string `bson:"status"`
	} `bson:"pnp"`
}

type checkinBoardingData struct {
	TicketID        string
	DateOfDeparture string
	TimeOfDeparture string
	Nopol           string
	POID            string
	NamaPO          string
	DestinationID   *int
	Tujuan          string
	PassengerName   string
	PassengerGender *string
	UserID          *string
	PembelianDari   string
	Source          string
}

type terminalBoardingpassResponse struct {
	ID               int64   `json:"id"`
	Noreg            string  `json:"noreg"`
	TicketID         string  `json:"ticket_id"`
	IDPO             string  `json:"id_po"`
	IDTujuan         *int    `json:"id_tujuan"`
	Nopol            string  `json:"nopol"`
	TanggalBerangkat string  `json:"tanggal_berangkat"`
	JamBerangkat     string  `json:"jam_berangkat"`
	NamaPNP          string  `json:"nama_pnp"`
	SexPNP           *string `json:"sex_pnp"`
	Fungsi           string  `json:"fungsi"`
	IDUser           *string `json:"id_user"`
	PembelianDari    *string `json:"pembelian_dari"`
	NamaPO           string  `json:"nama_po"`
	Tujuan           string  `json:"tujuan"`
	PrintURL         string  `json:"print_url"`
}

func CheckinMandiri(c *gin.Context) {
	var req request.CheckinMandiriRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "body tidak valid: " + err.Error()})
		return
	}

	ticketID := strings.TrimSpace(req.TicketID)
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": false, "message": "ticket_id wajib diisi"})
		return
	}

	if db.MongoClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "koneksi mongodb belum siap"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	collection := db.MongoClient.
		Database(config.C.MongoDatabase).
		Collection(config.C.MongoBoardingCollection)
	boardingpassCollection := db.MongoClient.
		Database(config.C.MongoDatabase).
		Collection(config.C.MongoBoardingpassCollection)

	tx, err := db.DB.Begin()
	if err != nil {
		logger.Error("begin checkin mandiri transaction failed ticket_id=%s err=%v", ticketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal memulai transaction"})
		return
	}
	defer tx.Rollback()

	existingBoardingpass, err := getExistingTodayBoardingpass(ctx, tx, ticketID)
	if err != nil {
		logger.Error("check existing boardingpass failed ticket_id=%s err=%v", ticketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal memeriksa boardingpass yang ada"})
		return
	}
	if existingBoardingpass != nil {
		existingBoardingpass.PrintURL = buildExistingPrintURL(c, existingBoardingpass)
		if existingBoardingpass.Fungsi == "checkin" {
			c.JSON(http.StatusOK, gin.H{"status": true, "message": "boardingpass sudah checkin hari ini", "data": existingBoardingpass})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"status": false, "message": "boardingpass dengan ticket_id ini sudah ada hari ini", "data": existingBoardingpass})
		return
	}

	boarding, fromMongo, err := getCheckinBoardingData(ctx, tx, collection, ticketID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"status": false, "message": checkinNotFoundMessage()})
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"status": false, "message": "request damri timeout"})
			return
		}
		logger.Error("get checkin data failed ticket_id=%s err=%v", ticketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Data checkin tidak ditemukan", "error": err.Error()})
		return
	}

	noreg, err := generateUniqueNoreg()
	if err != nil {
		logger.Error("generate noreg failed ticket_id=%s err=%v", ticketID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal membuat noreg"})
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
			id_user,
			pembelian_dari,
			created,
			updated
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'checkin', 0, 0, ?, ?, NOW(), NOW())
	`,
		noreg,
		boarding.TicketID,
		boarding.POID,
		boarding.DestinationID,
		boarding.Nopol,
		boarding.DateOfDeparture,
		boarding.TimeOfDeparture,
		boarding.PassengerName,
		boarding.PassengerGender,
		boarding.UserID,
		boarding.PembelianDari,
	)
	if err != nil {
		logger.Error("insert checkin mandiri failed ticket_id=%s err=%v", ticketID, err)
		if fromMongo {
			rollbackMongoCheckin(ctx, collection, ticketID)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal insert boardingpass"})
		return
	}

	insertedID, _ := result.LastInsertId()

	if err := insertMongoBoardingpass(ctx, boardingpassCollection, insertedID, noreg, boarding); err != nil {
		logger.Error("insert mongo boardingpass failed ticket_id=%s noreg=%s err=%v", ticketID, noreg, err)
		if fromMongo {
			rollbackMongoCheckin(ctx, collection, ticketID)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal insert boardingpass mongodb"})
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("commit checkin mandiri failed ticket_id=%s err=%v", ticketID, err)
		deleteMongoBoardingpass(ctx, boardingpassCollection, noreg)
		if fromMongo {
			rollbackMongoCheckin(ctx, collection, ticketID)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": false, "message": "gagal commit transaction"})
		return
	}

	logger.Info("CHECKIN MANDIRI SUCCESS ticket_id=%s boardingpass_id=%d", ticketID, insertedID)

	printURL := buildPrintURL(c, noreg, boarding)

	c.JSON(http.StatusCreated, gin.H{
		"status":         true,
		"id":             insertedID,
		"noreg":          noreg,
		"ticket_id":      ticketID,
		"source":         boarding.Source,
		"pembelian_dari": boarding.PembelianDari,
		"print_url":      printURL,
		"message":        "checkin mandiri berhasil",
	})
}

func checkinNotFoundMessage() string {
	if config.C.EnableDamriFallback {
		return "ticket_id tidak ditemukan di database maupun damri"
	}

	return "ticket_id tidak ditemukan"
}

func getExistingTodayBoardingpass(ctx context.Context, tx *sql.Tx, ticketID string) (*terminalBoardingpassResponse, error) {
	var data terminalBoardingpassResponse
	var idTujuan sql.NullInt64
	var sexPNP sql.NullString
	var idUser sql.NullString
	var pembelianDari sql.NullString

	err := tx.QueryRowContext(ctx, `
		SELECT
			id,
			noreg,
			COALESCE(ticket_id, ''),
			COALESCE(id_po, ''),
			id_tujuan,
			COALESCE(nopol, ''),
			DATE_FORMAT(tanggal_berangkat, '%Y-%m-%d'),
			TIME_FORMAT(jam_berangkat, '%H:%i:%s'),
			COALESCE(nama_pnp, ''),
			sex_pnp,
			COALESCE(fungsi, ''),
			id_user,
			pembelian_dari,
			CASE
				WHEN id_po = 'PO028' THEN 'Damri'
				ELSE COALESCE(NULLIF(id_po, ''), '-')
			END,
			COALESCE((
				SELECT tujuan
				FROM tbl_tujuan
				WHERE tbl_tujuan.id_tujuan = terminal_boardingpass.id_tujuan
				LIMIT 1
			), '-')
		FROM terminal_boardingpass
		WHERE ticket_id = ?
		AND DATE(tanggal_berangkat) = CURDATE()
		AND deleted IS NULL
		ORDER BY id DESC
		LIMIT 1
	`, ticketID).Scan(
		&data.ID,
		&data.Noreg,
		&data.TicketID,
		&data.IDPO,
		&idTujuan,
		&data.Nopol,
		&data.TanggalBerangkat,
		&data.JamBerangkat,
		&data.NamaPNP,
		&sexPNP,
		&data.Fungsi,
		&idUser,
		&pembelianDari,
		&data.NamaPO,
		&data.Tujuan,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if idTujuan.Valid {
		value := int(idTujuan.Int64)
		data.IDTujuan = &value
	}
	if sexPNP.Valid {
		data.SexPNP = &sexPNP.String
	}
	if idUser.Valid {
		data.IDUser = &idUser.String
	}
	if pembelianDari.Valid {
		data.PembelianDari = &pembelianDari.String
	}

	return &data, nil
}

func buildPrintURL(c *gin.Context, noreg string, boarding *checkinBoardingData) string {
	values := url.Values{}
	values.Set("ticket_id", boarding.TicketID)
	values.Set("nama_pnp", boarding.PassengerName)
	values.Set("nama_po", boarding.NamaPO)
	values.Set("tujuan", boarding.Tujuan)
	values.Set("tanggal_berangkat", boarding.DateOfDeparture)
	values.Set("jam_berangkat", boarding.TimeOfDeparture)
	values.Set("autoprint", "1")

	return requestBaseURL(c) + "/boarding/print/" + url.PathEscape(noreg) + "?" + values.Encode()
}

func buildExistingPrintURL(c *gin.Context, boarding *terminalBoardingpassResponse) string {
	values := url.Values{}
	values.Set("ticket_id", boarding.TicketID)
	values.Set("nama_pnp", boarding.NamaPNP)
	values.Set("nama_po", boarding.NamaPO)
	values.Set("tujuan", boarding.Tujuan)
	values.Set("tanggal_berangkat", boarding.TanggalBerangkat)
	values.Set("jam_berangkat", boarding.JamBerangkat)
	values.Set("autoprint", "1")

	return requestBaseURL(c) + "/boarding/print/" + url.PathEscape(boarding.Noreg) + "?" + values.Encode()
}

func requestBaseURL(c *gin.Context) string {
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}

	proto := c.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	return proto + "://" + host
}

func generateUniqueNoreg() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return "CM" + strings.ToUpper(hex.EncodeToString(randomBytes)), nil
}

func insertMongoBoardingpass(ctx context.Context, collection *mongo.Collection, mysqlID int64, noreg string, boarding *checkinBoardingData) error {
	now := time.Now()
	doc := bson.M{
		"mysql_id":          mysqlID,
		"noreg":             noreg,
		"ticket_id":         boarding.TicketID,
		"date_of_departure": boarding.DateOfDeparture,
		"time_of_departure": boarding.TimeOfDeparture,
		"nopol":             boarding.Nopol,
		"po_id":             boarding.POID,
		"nama_po":           boarding.NamaPO,
		"destination_id":    boarding.DestinationID,
		"tujuan":            boarding.Tujuan,
		"pnp": []bson.M{
			{
				"name":   boarding.PassengerName,
				"gender": boarding.PassengerGender,
				"status": "checkin",
			},
		},
		"fungsi":         "checkin",
		"status":         "checkin",
		"id_user":        boarding.UserID,
		"retribusi":      0,
		"count_scan":     0,
		"pembelian_dari": boarding.PembelianDari,
		"source":         boarding.Source,
		"createdAt":      now,
		"updatedAt":      now,
	}

	_, err := collection.InsertOne(ctx, doc)
	return err
}

func deleteMongoBoardingpass(ctx context.Context, collection *mongo.Collection, noreg string) {
	_, err := collection.DeleteOne(ctx, bson.M{"noreg": noreg})
	if err != nil {
		logger.Error("delete mongo boardingpass failed noreg=%s err=%v", noreg, err)
	}
}

func getCheckinBoardingData(ctx context.Context, tx *sql.Tx, collection *mongo.Collection, ticketID string) (*checkinBoardingData, bool, error) {
	checkinAt := time.Now()
	var mongoData mongoBoarding
	err := collection.FindOneAndUpdate(ctx,
		mongoBoardingFilter(ticketID),
		bson.M{
			"$set": bson.M{
				"status":    "checkin",
				"updatedAt": checkinAt,
			},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&mongoData)
	if err == nil {
		return mongoBoardingToCheckinData(mongoData), true, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, err
	}

	if !config.C.EnableDamriFallback {
		return nil, false, mongo.ErrNoDocuments
	}

	damriData, err := getDamriCheckinData(ctx, tx, ticketID)
	if err != nil {
		if errors.Is(err, client.ErrDamriTicketNotFound) {
			return nil, false, mongo.ErrNoDocuments
		}
		return nil, false, err
	}

	return damriData, false, nil
}

func mongoBoardingFilter(ticketID string) bson.M {
	filter := bson.M{
		"ticket_id": ticketID,
		"status":    "active",
	}

	if len(config.C.IgnoredMongoUserIDs) > 0 {
		filter["user_id"] = bson.M{
			"$nin": config.C.IgnoredMongoUserIDs,
		}
	}

	return filter
}

func mongoBoardingToCheckinData(boarding mongoBoarding) *checkinBoardingData {
	passengerName := ""
	var passengerGender *string
	if len(boarding.PNP) > 0 {
		passengerName = boarding.PNP[0].Name
		passengerGender = boarding.PNP[0].Gender
	}

	var userID *string
	if boarding.UserID != nil {
		value := strconv.Itoa(*boarding.UserID)
		userID = &value
	}

	var destinationID *int
	if boarding.DestinationID != 0 {
		value := int(boarding.DestinationID)
		destinationID = &value
	}

	return &checkinBoardingData{
		TicketID:        boarding.TicketID,
		DateOfDeparture: boarding.DateOfDeparture,
		TimeOfDeparture: boarding.TimeOfDeparture,
		Nopol:           boarding.Nopol,
		POID:            boarding.POID,
		NamaPO:          mongoNamaPO(boarding),
		DestinationID:   destinationID,
		Tujuan:          mongoTujuan(boarding),
		PassengerName:   passengerName,
		PassengerGender: passengerGender,
		UserID:          userID,
		PembelianDari:   mongoPembelianDari(boarding),
		Source:          "mongodb",
	}
}

func mongoNamaPO(boarding mongoBoarding) string {
	if value := strings.TrimSpace(boarding.PO.NamaPO); value != "" {
		return value
	}
	if value := strings.TrimSpace(boarding.PO.NamaSingkat); value != "" {
		return value
	}
	if value := strings.TrimSpace(boarding.POID); value != "" {
		return value
	}

	return "-"
}

func mongoTujuan(boarding mongoBoarding) string {
	if value := strings.TrimSpace(boarding.Destination.Tujuan); value != "" {
		return value
	}

	return "-"
}

func mongoPembelianDari(boarding mongoBoarding) string {
	username := strings.TrimSpace(boarding.User.Username)
	if username != "" {
		return username
	}

	return "Checkin Mandiri"
}

func getDamriCheckinData(ctx context.Context, tx *sql.Tx, ticketID string) (*checkinBoardingData, error) {
	if config.C.DamriUsername == "" || config.C.DamriPassword == "" {
		return nil, errors.New("config damri belum lengkap")
	}

	timeout := time.Duration(config.C.DamriTimeoutSeconds) * time.Second
	damriCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	scanData, err := getDamriClient(timeout).ScanTicket(damriCtx, ticketID)
	if err != nil {
		logger.Warning("damri fallback failed ticket_id=%s err=%v", ticketID, err)
		return nil, err
	}

	destinationID, err := getDamriDestinationID(tx, scanData)
	if err != nil {
		return nil, err
	}

	return &checkinBoardingData{
		TicketID:        scanData.TicketID,
		DateOfDeparture: scanData.DateOfDeparture,
		TimeOfDeparture: scanData.TimeOfDeparture,
		Nopol:           scanData.Nopol,
		POID:            "PO028",
		NamaPO:          "Damri",
		DestinationID:   destinationID,
		Tujuan:          scanData.DestinationName,
		PassengerName:   scanData.PassengerName,
		PembelianDari:   "Checkin Mandiri",
		Source:          "damri",
	}, nil
}

func getDamriClient(timeout time.Duration) *client.DamriClient {
	damriClientMu.Lock()
	defer damriClientMu.Unlock()

	if damriClientCached == nil {
		damriClientCached = client.NewDamriClient(
			config.C.DamriBaseURL,
			config.C.DamriUsername,
			config.C.DamriPassword,
			timeout,
		)
	}

	return damriClientCached
}

func getDamriDestinationID(tx *sql.Tx, data *client.DamriScanData) (*int, error) {
	if fallback, ok := damriDestinationFallback[strings.ToUpper(data.DestinationCode)]; ok {
		return &fallback, nil
	}

	name := cleanDamriDestinationName(data.DestinationName)
	return getIDTujuanFromTujuanTx(tx, name)
}

func cleanDamriDestinationName(name string) string {
	replacer := strings.NewReplacer(
		"Terminal", "",
		"terminal", "",
		"Pool Damri", "",
		"pool damri", "",
		"POOL DAMRI", "",
	)

	return strings.TrimSpace(replacer.Replace(name))
}

var damriDestinationFallback = map[string]int{
	"YOG": 90,
	"BWI": 374,
	"BUN": 718,
	"TWO": 430,
}

func rollbackMongoCheckin(ctx context.Context, collection *mongo.Collection, ticketID string) {
	_, err := collection.UpdateOne(ctx,
		bson.M{
			"ticket_id": ticketID,
			"status":    "checkin",
		},
		bson.M{
			"$set": bson.M{
				"status":    "active",
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		logger.Error("rollback mongo checkin failed ticket_id=%s err=%v", ticketID, err)
	}
}
