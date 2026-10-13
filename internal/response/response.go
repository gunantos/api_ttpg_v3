package response

type GateResponse struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
}

type BoardingpassResponse struct {
	ID               int    `json:"id"`
	Noreg            string `json:"noreg"`
	IDPO             string `json:"id_po"`
	IDTujuan         int    `json:"id_tujuan"`
	Nopol            string `json:"nopol"`
	TanggalBerangkat string `json:"tanggal_berangkat"`
	JamBerangkat     string `json:"jam_berangkat"`
	NamaPnp          string `json:"nama_pnp"`
	Fungsi           string `json:"fungsi"`
	CountScan        int    `json:"count_scan"`
	Created          string `json:"created"`
}

type ListBoardingpassResponse struct {
	Data  []BoardingpassResponse `json:"data"`
	Total int                    `json:"total"`
}
