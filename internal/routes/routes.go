package routes

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/activity"
	"github.com/andtho89/ttpg/internal/admin"
	"github.com/andtho89/ttpg/internal/docs"
	"github.com/andtho89/ttpg/internal/handler"
	"github.com/andtho89/ttpg/internal/login"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/andtho89/ttpg/internal/pkg/response"
	"github.com/andtho89/ttpg/internal/platform/auth"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, mainDB *gorm.DB, mongoClient *mongo.Client) {
	boardingCol := mongoClient.Database("ttpg").Collection("boarding")

	r.GET("/boarding/print/:code", handler.PrintBoardingpass)
	r.GET("/boardingpass/print/:code", handler.PrintBoardingpass)
	r.GET("/ws", handler.HandleWS)
	r.GET("/docs", docs.Serve)
	r.GET("/activity", activity.Serve)
	r.GET("/login", login.Serve)
	r.GET("/admin", admin.Serve)
	r.GET("/penginapan/payments/:id/receipt", handler.PaymentReceipt)
	r.POST("/penginapan/webhooks/midtrans", handler.WebhookMidtrans)
	r.POST("/penginapan/webhooks/xendit", handler.WebhookXendit)

	apiKeyAuth := r.Group("", auth.AuthMiddleware())
	apiKeyAuth.GET("/activity/logs", handler.ListActivityLogs)
	apiKeyAuth.GET("/vending/config", handler.VendingConfig)
	apiKeyAuth.POST("/vending/device/status", handler.VendingDeviceStatus)
	apiKeyAuth.GET("/vending/media", handler.VendingMedia)
	apiKeyAuth.POST("/gate", handler.Gate)
	apiKeyAuth.POST("/checkin-mandiri", handler.CheckinMandiri)
	apiKeyAuth.GET("/boardingpass", handler.List)
	apiKeyAuth.GET("/boardingpass/:id", handler.GetByID)
	apiKeyAuth.POST("/boardingpass", handler.Create)
	apiKeyAuth.PUT("/boardingpass/:id", handler.Update)
	apiKeyAuth.DELETE("/boardingpass/:id", handler.Delete)
	apiKeyAuth.POST("/boardingpass/:id/ktp", handler.UploadBoardingpassImage)
	apiKeyAuth.GET("/boardingpass/:id/ktp", handler.GetBoardingpassImage)
	apiKeyAuth.DELETE("/boardingpass/:id/ktp", handler.DeleteBoardingpassImage)
	apiKeyAuth.POST("/boardingpass/:id/tiket", handler.UploadBoardingpassImage)
	apiKeyAuth.GET("/boardingpass/:id/tiket", handler.GetBoardingpassImage)
	apiKeyAuth.DELETE("/boardingpass/:id/tiket", handler.DeleteBoardingpassImage)
	r.POST("/auth", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{
				{Code: "VALIDATION_ERROR", Detail: err.Error()},
			})
			return
		}

		u, err := auth.Login(req.Username, req.Password)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "Username atau password salah", nil)
			return
		}

		role := auth.RoleFromUser(u)
		token, _ := auth.GenerateToken(u.ID, role, auth.ParsePOIDs(u.PoIDOrEmpty()))
		response.Success(c, http.StatusOK, "Login berhasil", gin.H{
			"token":   token,
			"user_id": u.ID,
			"role":    role,
			"expire":  time.Now().Add(auth.TokenExpiry).Format(time.RFC3339),
		}, nil)
	})

	api := r.Group("")
	api.Use(auth.AuthMiddleware())
	api.GET("/tujuan", func(c *gin.Context) {
		var data []models.Tujuan
		if err := mainDB.Order("id_tujuan asc").Find(&data).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal mengambil data tujuan", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		response.Success(c, http.StatusOK, "Data tujuan ditemukan", data, nil)
	})

	api.GET("/po", func(c *gin.Context) {
		var data []struct {
			IDPO   string `json:"id_po"`
			NamaPO string `json:"nama_po"`
		}
		if err := mainDB.Table("tbl_po").Select("id_po, nama_po").Order("id_po asc").Scan(&data).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal mengambil data po", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		response.Success(c, http.StatusOK, "Data po ditemukan", data, nil)
	})

	api.GET("/bus", func(c *gin.Context) {
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			q = strings.TrimSpace(c.Query("search"))
		}
		limit := 20
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 100 {
				response.Error(c, http.StatusBadRequest, "limit harus 1-100", []response.ErrorItem{{Field: "limit", Code: "VALIDATION_ERROR", Detail: "limit harus 1-100"}})
				return
			}
			limit = n
		}
		var data []struct {
			IDBus          int    `json:"id_bus" gorm:"column:id_bus"`
			Kode           string `json:"kode" gorm:"column:kode"`
			Nopol          string `json:"nopol" gorm:"column:nopol"`
			IDPO           string `json:"id_po" gorm:"column:id_po"`
			NamaPO         string `json:"nama_po" gorm:"column:nama_po"`
			NmPerusahaan   string `json:"nm_perusahaan" gorm:"column:nm_perusahaan"`
			Trayek1        string `json:"trayek1" gorm:"column:trayek1"`
			Trayek2        string `json:"trayek2" gorm:"column:trayek2"`
			JenisBus       string `json:"jenis_bus" gorm:"column:jenis_bus"`
			Pnp            int    `json:"pnp" gorm:"column:pnp"`
			Status         string `json:"status" gorm:"column:status"`
			TrayekMainID   int    `json:"trayek_main_id" gorm:"column:trayek_main_id"`
			TrayekKode     string `json:"trayek_kode" gorm:"column:trayek_kode"`
			AsalID         int    `json:"asal_id" gorm:"column:asal_id"`
			Asal           string `json:"asal" gorm:"column:asal"`
			TujuanID       int    `json:"tujuan_id" gorm:"column:tujuan_id"`
			Tujuan         string `json:"tujuan" gorm:"column:tujuan"`
			Rute           string `json:"rute" gorm:"column:rute"`
		}
		dbq := mainDB.Table("tbl_bus b").
			Select("b.id_bus, b.kode, b.nopol, b.id_po, COALESCE(p.nama_po,'') AS nama_po, b.nm_perusahaan, b.trayek1, b.trayek2, b.jenis_bus, b.pnp, b.status, b.trayek_main_id, " +
				"COALESCE(t.kode_trayek,'') AS trayek_kode, COALESCE(t.asal,0) AS asal_id, COALESCE(ta.tujuan,'') AS asal, " +
				"COALESCE(t.tujuan,0) AS tujuan_id, COALESCE(tu.tujuan,'') AS tujuan, COALESCE(t.rute,'') AS rute").
			Joins("LEFT JOIN tbl_po p ON p.id_po = b.id_po").
			Joins("LEFT JOIN terminal_trayek_main t ON t.id = b.trayek_main_id").
			Joins("LEFT JOIN tbl_tujuan ta ON ta.id_tujuan = t.asal").
			Joins("LEFT JOIN tbl_tujuan tu ON tu.id_tujuan = t.tujuan")
		if v := strings.TrimSpace(c.Query("id_po")); v != "" {
			dbq = dbq.Where("b.id_po = ?", v)
		}
		if v := strings.TrimSpace(c.Query("status")); v != "" {
			dbq = dbq.Where("b.status = ?", v)
		}
		if q != "" {
			like := "%" + q + "%"
			dbq = dbq.Where("b.kode LIKE ? OR b.nopol LIKE ? OR b.trayek1 LIKE ? OR b.trayek2 LIKE ? OR b.nm_perusahaan LIKE ? OR b.id_po LIKE ? OR t.kode_trayek LIKE ? OR ta.tujuan LIKE ? OR tu.tujuan LIKE ? OR t.rute LIKE ?", like, like, like, like, like, like, like, like, like, like)
		}
		if err := dbq.Order("b.nopol asc, b.id_bus asc").Limit(limit).Find(&data).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal mengambil data bus", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		response.Success(c, http.StatusOK, "Data bus ditemukan", data, nil)
	})

	api.GET("/boarding", func(c *gin.Context) {
		poIDs := c.MustGet("po_ids").([]string)
		filter := bson.M{}
		if len(poIDs) > 0 {
			filter["po_id"] = bson.M{"$in": poIDs}
		}
		if v := c.Query("po_id"); v != "" {
			filter["po_id"] = v
		}
		if v := c.Query("ticket_id"); v != "" {
			filter["ticket_id"] = v
		}
		if v := c.Query("destination_id"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "destination_id tidak valid", []response.ErrorItem{{Field: "destination_id", Code: "VALIDATION_ERROR", Detail: err.Error()}})
				return
			}
			filter["$or"] = []bson.M{
				{"destination_id": n},
				{"destination_id": strconv.Itoa(n)},
				{"destination_id.id_tujuan": n},
			}
		}
		limit := 100
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 1000 {
				response.Error(c, http.StatusBadRequest, "limit harus 1-1000", []response.ErrorItem{{Field: "limit", Code: "VALIDATION_ERROR", Detail: "limit harus 1-1000"}})
				return
			}
			limit = n
		}
		offset := 0
		if v := strings.TrimSpace(c.Query("offset")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				response.Error(c, http.StatusBadRequest, "offset tidak valid", []response.ErrorItem{{Field: "offset", Code: "VALIDATION_ERROR", Detail: "offset tidak valid"}})
				return
			}
			offset = n
		}
		total, err := boardingCol.CountDocuments(context.TODO(), filter)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menghitung data boarding", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		cur, err := boardingCol.Find(context.TODO(), filter, options.Find().SetSkip(int64(offset)).SetLimit(int64(limit)))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal mengambil data boarding", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		defer cur.Close(context.TODO())
		data := []models.BoardingDoc{}
		if err := cur.All(context.TODO(), &data); err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal decode data", []response.ErrorItem{{Code: "DECODE_ERROR", Detail: err.Error()}})
			return
		}
		pagination := &response.Pagination{
			Page:       offset/limit + 1,
			Limit:      limit,
			Total:      int(total),
			TotalPages: (int(total) + limit - 1) / limit,
		}
		response.Success(c, http.StatusOK, "Data boarding ditemukan", data, pagination)
	})

	api.POST("/boarding", func(c *gin.Context) {
		var req struct {
			POID            string `json:"po_id"`
			TicketID        string `json:"ticket_id" binding:"required"`
			Nopol           string `json:"nopol"`
			DestinationID   int    `json:"destination_id" binding:"required"`
			DateOfDeparture string `json:"date_of_departure" binding:"required"`
			TimeOfDeparture string `json:"time_of_departure" binding:"required"`
			NIK             string `json:"nik"`
			Name            string `json:"name" binding:"required"`
			Gender          string `json:"gender"`
			Address         string `json:"address"`
			City            string `json:"city"`
			Province        string `json:"province"`
			Telp            string `json:"telp"`
			Email           string `json:"email"`
		}
		if err := c.BindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "Data tidak valid", []response.ErrorItem{{Code: "VALIDATION_ERROR", Detail: err.Error()}})
			return
		}

		poIDs := c.MustGet("po_ids").([]string)
		if len(poIDs) > 0 {
			if req.POID == "" && len(poIDs) == 1 {
				req.POID = poIDs[0]
			}
			allowed := false
			for _, p := range poIDs {
				if p == req.POID {
					allowed = true
					break
				}
			}
			if !allowed {
				response.Error(c, http.StatusForbidden, "po_id tidak diizinkan untuk user ini", nil)
				return
			}
		}
		if req.POID == "" {
			response.Error(c, http.StatusBadRequest, "po_id wajib diisi", []response.ErrorItem{{Field: "po_id", Code: "REQUIRED", Detail: "po_id wajib diisi"}})
			return
		}

		names := splitMulti(req.Name)
		genders := splitMulti(req.Gender)
		now := time.Now()
		pnp := []models.PnP{}
		for i, n := range names {
			g := ""
			if len(genders) == 1 {
				g = genders[0]
			} else if i < len(genders) {
				g = genders[i]
			}
			pnp = append(pnp, models.PnP{Name: n, Gender: g, Status: nil, CreatedAt: now, UpdatedAt: now})
		}

		doc := models.BoardingDoc{
			BoardingID:      now.Unix(),
			TicketID:        req.TicketID,
			DateOfDeparture: req.DateOfDeparture,
			TimeOfDeparture: req.TimeOfDeparture,
			Nopol:           req.Nopol,
			POID:            req.POID,
			DestinationID:   models.FlexInt(req.DestinationID),
			PNP:             pnp,
			UserID:          c.MustGet("user_id").(uint),
			Status:          "active",
			NIK:             req.NIK,
			Address:         req.Address,
			City:            req.City,
			Province:        req.Province,
			Telp:            req.Telp,
			Email:           req.Email,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		res, err := boardingCol.InsertOne(context.TODO(), doc)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal menyimpan boarding", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		doc.ID = res.InsertedID.(bson.ObjectID)
		response.Success(c, http.StatusCreated, "Boarding berhasil dibuat", doc, nil)
	})

	api.PUT("/boarding/:id", func(c *gin.Context) {
		id, err := bson.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
			return
		}
		var req struct {
			Nopol           *string `json:"nopol"`
			Status          *string `json:"status"`
			DateOfDeparture *string `json:"date_of_departure"`
			TimeOfDeparture *string `json:"time_of_departure"`
			DestinationID   *int    `json:"destination_id"`
		}
		if err := c.BindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "Data tidak valid", nil)
			return
		}
		poIDs := c.MustGet("po_ids").([]string)
		var existing models.BoardingDoc
		if err := boardingCol.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&existing); err != nil {
			response.Error(c, http.StatusNotFound, "Boarding tidak ditemukan", nil)
			return
		}
		if len(poIDs) > 0 {
			allowed := false
			for _, p := range poIDs {
				if p == existing.POID {
					allowed = true
					break
				}
			}
			if !allowed {
				response.Error(c, http.StatusForbidden, "po_id tidak diizinkan", nil)
				return
			}
		}
		set := bson.M{"updatedAt": time.Now()}
		if req.Nopol != nil {
			set["nopol"] = *req.Nopol
		}
		if req.Status != nil {
			set["status"] = *req.Status
		}
		if req.DateOfDeparture != nil {
			set["date_of_departure"] = *req.DateOfDeparture
		}
		if req.TimeOfDeparture != nil {
			set["time_of_departure"] = *req.TimeOfDeparture
		}
		if req.DestinationID != nil {
			set["destination_id"] = *req.DestinationID
		}
		_, err = boardingCol.UpdateByID(context.TODO(), id, bson.M{"$set": set})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal update boarding", []response.ErrorItem{{Code: "DB_ERROR", Detail: err.Error()}})
			return
		}
		response.Success(c, http.StatusOK, "Boarding berhasil diupdate", nil, nil)
	})

	api.DELETE("/boarding/:id", func(c *gin.Context) {
		id, err := bson.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, "id tidak valid", nil)
			return
		}
		poIDs := c.MustGet("po_ids").([]string)
		var existing models.BoardingDoc
		if err := boardingCol.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&existing); err != nil {
			response.Error(c, http.StatusNotFound, "Boarding tidak ditemukan", nil)
			return
		}
		if len(poIDs) > 0 {
			allowed := false
			for _, p := range poIDs {
				if p == existing.POID {
					allowed = true
					break
				}
			}
			if !allowed {
				response.Error(c, http.StatusForbidden, "po_id tidak diizinkan", nil)
				return
			}
		}
		if _, err := boardingCol.DeleteOne(context.TODO(), bson.M{"_id": id}); err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal hapus boarding", nil)
			return
		}
		response.Success(c, http.StatusOK, "Boarding berhasil dihapus", nil, nil)
	})

	peng := r.Group("/penginapan")
	peng.Use(auth.AuthMiddleware())

	peng.GET("/configs", handler.ListConfigs)
	peng.GET("/configs/key/:key", handler.GetConfigByKey)
	peng.POST("/configs", handler.CreateConfig)
	peng.PUT("/configs/:id", handler.UpdateConfig)
	peng.DELETE("/configs/:id", handler.DeleteConfig)

	peng.GET("/qris", handler.GetQRISPenginapan)

	peng.GET("/payment-methods", handler.ListPaymentMethods)
	peng.POST("/payment-methods", handler.CreatePaymentMethod)
	peng.PUT("/payment-methods/:id", handler.UpdatePaymentMethod)
	peng.DELETE("/payment-methods/:id", handler.DeletePaymentMethod)

	peng.GET("/purchase-sources", handler.ListPurchaseSources)
	peng.POST("/purchase-sources", handler.CreatePurchaseSource)
	peng.PUT("/purchase-sources/:id", handler.UpdatePurchaseSource)
	peng.DELETE("/purchase-sources/:id", handler.DeletePurchaseSource)

	peng.GET("/locations", handler.ListLocations)
	peng.GET("/locations/:id", handler.GetLocation)
	peng.POST("/locations", handler.CreateLocation)
	peng.PUT("/locations/:id", handler.UpdateLocation)
	peng.DELETE("/locations/:id", handler.DeleteLocation)

	peng.GET("/rooms", handler.ListRooms)
	peng.GET("/rooms/:id", handler.GetRoom)
	peng.POST("/rooms", handler.CreateRoom)
	peng.PUT("/rooms/:id", handler.UpdateRoom)
	peng.PUT("/rooms/:id/status", handler.UpdateRoomStatus)
	peng.DELETE("/rooms/:id", handler.DeleteRoom)

	peng.GET("/room-layouts", handler.ListRoomLayouts)
	peng.POST("/room-layouts", handler.CreateRoomLayout)
	peng.PUT("/room-layouts/:id", handler.UpdateRoomLayout)
	peng.DELETE("/room-layouts/:id", handler.DeleteRoomLayout)

	peng.GET("/guests", handler.ListGuests)
	peng.GET("/guests/:id", handler.GetGuest)
	peng.POST("/guests", handler.CheckinPenginapan)
	peng.PUT("/guests/:id", handler.UpdateGuestPenginapan)
	peng.POST("/guests/:id/checkout", handler.CheckoutPenginapan)
	peng.POST("/guests/:id/pindah", handler.PindahKamarPenginapan)
	peng.GET("/guests/:id/histories", handler.ListGuestHistories)
	peng.POST("/guests/:id/ktp", handler.UploadGuestKTP)
	peng.GET("/guests/:id/ktp", handler.GetGuestKTP)
	peng.DELETE("/guests/:id/ktp", handler.DeleteGuestKTP)

	peng.GET("/payments", handler.ListPayments)
	peng.GET("/payments/:id", handler.GetPayment)
	peng.POST("/payments", handler.CreatePaymentPenginapan)
	peng.POST("/payments/:id/pay", handler.PayWithPayment)
	peng.PUT("/payments/:id", handler.UpdatePaymentPenginapan)
	peng.DELETE("/payments/:id", handler.VoidPaymentPenginapan)
	peng.GET("/payment-gateway", handler.GetPaymentGateway)

	peng.GET("/room-histories", handler.ListRoomHistories)
}

func splitMulti(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ";", ",")
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
