package models

type PO struct {
	IDPO        int    `gorm:"primaryKey;column:id_po" json:"id_po"`
	NamaPO      string `json:"nama_po"`
	NamaSingkat string `json:"nama_singkat"`
	PTPO        string `json:"pt_po"`
	AlmtPO      string `json:"almt_po"`
	TlpPO       string `json:"tlp_po"`
	GambarPO    string `json:"gambar_po"`
	Email       string `json:"email"`
	KodePerusahaan string `json:"kode_perusahaan"`
	NoSK        string `json:"no_sk"`
	ExpSK       string `json:"exp_sk"`
	IsActive    int    `json:"is_active"`
	CreateDate  string `json:"create_date"`
	UpdateDate  string `json:"update_date"`
}

func (PO) TableName() string {
	return "tbl_po"
}
