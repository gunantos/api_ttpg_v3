package models

import "time"

type Tujuan struct {
	IDTujuan    int       `gorm:"primaryKey;column:id_tujuan" json:"id_tujuan"`
	Provinsi    string    `json:"provinsi"`
	Tujuan      string    `json:"tujuan"`
	Kawasan     string    `json:"kawasan"`
	KodeTujuan  string    `json:"kode_tujuan"`
	NamaTerminal string   `json:"nama_terminal"`
	CreateDate  time.Time `json:"create_date"`
	UpdateDate  time.Time `json:"update_date"`
}

func (Tujuan) TableName() string {
	return "tbl_tujuan"
}
