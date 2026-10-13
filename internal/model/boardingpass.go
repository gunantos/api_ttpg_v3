package model

import (
	"database/sql"
	"time"
)

type Boardingpass struct {
	ID               int            `json:"id"`
	Noreg            string         `json:"noreg"`
	TicketID         sql.NullString `json:"ticket_id"`
	IDPO             string         `json:"id_po"`
	IDTujuan         sql.NullInt64  `json:"id_tujuan"`
	Nopol            string         `json:"nopol"`
	TanggalBerangkat string         `json:"tanggal_berangkat"`
	JamBerangkat     string         `json:"jam_berangkat"`
	NamaPnp          string         `json:"nama_pnp"`
	SexPnp           sql.NullString `json:"sex_pnp"`
	Fungsi           sql.NullString `json:"fungsi"`
	GateID           sql.NullInt64  `json:"gate_id"`
	DeviceID         sql.NullString `json:"device_id"`
	TimeTab          sql.NullTime   `json:"time_tab"`
	IDUser           sql.NullString `json:"id_user"`
	Retribusi        int            `json:"retribusi"`
	CountScan        sql.NullInt64  `json:"count_scan"`
	PembelianDari    sql.NullString `json:"pembelian_dari"`
	Created          time.Time      `json:"created"`
	Updated          sql.NullTime   `json:"updated"`
	Deleted          sql.NullTime   `json:"deleted"`
}
