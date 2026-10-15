package auth

import (
	"strconv"
	"strings"
	"time"

	"github.com/andtho89/ttpg/internal/logger"
	"github.com/andtho89/ttpg/internal/models"
	"github.com/gin-gonic/gin"
)

func normalizeEndpoint(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	s = strings.TrimRight(s, "/")
	if strings.HasSuffix(s, "/index") {
		s = strings.TrimSuffix(s, "/index")
	}
	if s == "" {
		s = "/"
	}
	return s
}

// findApiURL mencari baris api_url untuk (method, path). Method rule '*' cocok semua.
// Controller dibandingkan dalam bentuk ternormalisasi ("/tujuan/index" == "/tujuan"),
// terhadap path request maupun pola route Gin (cth "/penginapan/rooms/:id").
func findApiURL(method, path, ginPattern string) (*models.TtpgApiURL, error) {
	var rows []models.TtpgApiURL
	if err := UsersDB.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	m := strings.ToLower(strings.TrimSpace(method))
	np := normalizeEndpoint(path)
	npp := normalizeEndpoint(ginPattern)
	for i := range rows {
		r := &rows[i]
		if r.Method != "*" && strings.ToLower(r.Method) != m {
			continue
		}
		rc := normalizeEndpoint(r.Controller)
		if rc != "" && (rc == np || rc == npp) {
			return r, nil
		}
	}
	return nil, nil
}

func schemeMismatch(required, actual string) (string, bool) {
	switch required {
	case "", "none":
		return "", true
	case "api_key":
		if actual == "api_key" {
			return "", true
		}
	case "bearer_token":
		if actual == "bearer" {
			return "", true
		}
	case "basic_auth":
		if actual == "basic" {
			return "", true
		}
	default:
		return "Skema wajib endpoint (" + required + ") tidak didukung", false
	}
	return "Endpoint ini mewajibkan skema " + required, false
}

func splitIDList(s string) map[int]bool {
	out := map[int]bool{}
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' }) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			out[n] = true
		}
	}
	return out
}

// accessAllowed mengecek izin user untuk endpoint (rule id) via tabel access + user.allowed.
func accessAllowed(u *models.TtpgUser, ruleID int) (string, bool) {
	if u.AllAccess == "y" {
		return "", true
	}
	allowed := splitIDList(u.AllowedOrEmpty())
	var acc models.TtpgAccess
	if err := UsersDB.Where("user_id = ?", u.ID).First(&acc).Error; err == nil {
		if acc.Block == "y" {
			return "Akses user diblokir", false
		}
		if acc.Allow != "y" {
			return "Akses user tidak diizinkan", false
		}
		for k := range splitIDList(acc.IDAPI) {
			allowed[k] = true
		}
	}
	if !allowed[ruleID] {
		return "Tidak memiliki akses ke endpoint ini", false
	}
	return "", true
}

var logCh chan models.TtpgLog

func logWorker() {
	for e := range logCh {
		if err := UsersDB.Create(&e).Error; err != nil {
			logger.Error("gagal tulis log ttpg_users: %v", err)
		}
	}
}

func enqueueLog(c *gin.Context, start time.Time, scheme, keyUsed string, identified bool) {
	if logCh == nil {
		return
	}
	uri := c.Request.URL.Path
	if len(uri) > 250 {
		uri = uri[:250]
	}
	params := logParams(c)
	if keyUsed == "" {
		keyUsed = ""
	} else if len(keyUsed) > 40 {
		keyUsed = keyUsed[:40]
	}
	authorized := "n"
	if identified {
		authorized = "y"
	}
	accessBy := scheme
	if accessBy == "" {
		accessBy = "none"
	}
	idUser := 0
	poStr := ""
	if identified {
		if v, ok := c.Get("user_id"); ok {
			if id, ok2 := v.(uint); ok2 {
				idUser = int(id)
			}
		}
	}
	if v, ok := c.Get("po_ids"); ok {
		if ids, ok2 := v.([]string); ok2 && len(ids) > 0 {
			poStr = strings.Join(ids, ";")
		}
	}
	sec := time.Since(start).Seconds()
	status := c.Writer.Status()
	if status == 0 {
		status = 200
	}
	e := models.TtpgLog{
		URI:          uri,
		Method:       c.Request.Method,
		Params:       &params,
		APIKey:       keyUsed,
		IPAddress:    c.ClientIP(),
		Time:         int(time.Now().Unix()),
		Rtime:        &sec,
		Authorized:   authorized,
		ResponseCode: status,
		IDUser:       idUser,
		PoID:         &poStr,
		AccessBy:     accessBy,
		Date:         time.Now(),
	}
	select {
	case logCh <- e:
	default:
	}
}
