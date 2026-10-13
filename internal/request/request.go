package request

type GateRequest struct {
	Noreg  string `json:"noreg" binding:"required"`
	GateID int    `json:"gate_id"`
}

type CreateBoardingpassRequest struct {
	Noreg            string  `json:"noreg" binding:"required"`
	TicketID         *string `json:"ticket_id"`
	IDPO             string  `json:"id_po"`
	IDTujuan         int     `json:"id_tujuan"`
	Nopol            string  `json:"nopol"`
	TanggalBerangkat string  `json:"tanggal_berangkat"` // YYYY-MM-DD
	JamBerangkat     string  `json:"jam_berangkat"`     // HH:MM:SS
	NamaPnp          string  `json:"nama_pnp"`
	SexPnp           *string `json:"sex_pnp"`
	Fungsi           string  `json:"fungsi"`
	Retribusi        int     `json:"retribusi"`
	PembelianDari    *string `json:"pembelian_dari"`
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
