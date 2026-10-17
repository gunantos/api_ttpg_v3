package request

type GateRequest struct {
	Noreg  string `json:"noreg" binding:"required"`
	GateID int    `json:"gate_id"`
}

type CreateBoardingpassRequest struct {
	Noreg            string  `json:"noreg" form:"noreg" binding:"required"`
	TicketID         *string `json:"ticket_id" form:"ticket_id"`
	IDPO             string  `json:"id_po" form:"id_po"`
	IDTujuan         int     `json:"id_tujuan" form:"id_tujuan"`
	Nopol            string  `json:"nopol" form:"nopol"`
	TanggalBerangkat string  `json:"tanggal_berangkat" form:"tanggal_berangkat"` // YYYY-MM-DD
	JamBerangkat     string  `json:"jam_berangkat" form:"jam_berangkat"`         // HH:MM:SS
	NamaPnp          string  `json:"nama_pnp" form:"nama_pnp"`
	SexPnp           *string `json:"sex_pnp" form:"sex_pnp"`
	Fungsi           string  `json:"fungsi" form:"fungsi"`
	Retribusi        int     `json:"retribusi" form:"retribusi"`
	PembelianDari    *string `json:"pembelian_dari" form:"pembelian_dari"`
}

type UpdateBoardingpassRequest struct {
	Fungsi    *string `json:"fungsi"`
	GateID    *string `json:"gate_id"`
	DeviceID  *string `json:"device_id"`
	TimeTab   *string `json:"time_tab"`
	CountScan *int    `json:"count_scan"`
}

type CheckinMandiriRequest struct {
	TicketID string `json:"ticket_id" form:"ticket_id"`
}
