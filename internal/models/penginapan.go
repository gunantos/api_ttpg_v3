package models

import "time"

type PenginapanConfig struct {
	IDConfigs    int       `gorm:"primaryKey;column:id_configs;type:int(11);autoIncrement" json:"id_configs"`
	KeyConfigs   string    `gorm:"column:key_configs;size:250;uniqueIndex" json:"key_configs"`
	ValueConfigs *string   `gorm:"column:value_configs;type:text" json:"value_configs"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PenginapanConfig) TableName() string { return "configs" }

type PaymentMethod struct {
	ID        int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	Code      string    `gorm:"column:code;size:30;uniqueIndex" json:"code"`
	Name      string    `gorm:"column:name;size:100" json:"name"`
	IsActive  bool      `gorm:"column:is_active;type:tinyint(1);default:1" json:"is_active"`
	SortOrder int       `gorm:"column:sort_order;type:int(11);default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PaymentMethod) TableName() string { return "payment_methods" }

type PurchaseSource struct {
	ID        int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	Code      string    `gorm:"column:code;size:30;uniqueIndex" json:"code"`
	Name      string    `gorm:"column:name;size:100" json:"name"`
	IsActive  bool      `gorm:"column:is_active;type:tinyint(1);default:1" json:"is_active"`
	SortOrder int       `gorm:"column:sort_order;type:int(11);default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PurchaseSource) TableName() string { return "purchase_sources" }

type PenginapanLocation struct {
	ID         int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	Name       string    `gorm:"column:name;size:100;not null" json:"name"`
	Keterangan string    `gorm:"column:keterangan;size:250;not null" json:"keterangan"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PenginapanLocation) TableName() string { return "locations" }

type PenginapanRoom struct {
	ID               int                 `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	LocationID       *int                `gorm:"column:location_id;type:int(11);index" json:"location_id"`
	Location         *PenginapanLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	RoomNumber       string              `gorm:"column:room_number;size:100;not null;uniqueIndex:idx_rooms_loc_number" json:"room_number"`
	RoomType         string              `gorm:"column:room_type;type:enum('pria','wanita');not null" json:"room_type"`
	Capacity         int                 `gorm:"column:capacity;type:int(11);not null;default:1" json:"capacity"`
	CurrentOccupants int                 `gorm:"column:current_occupants;type:int(11);not null;default:0" json:"current_occupants"`
	Status           string              `gorm:"column:status;type:enum('kosong','terisi','maintenance');not null;default:kosong" json:"status"`
	CreatedAt        time.Time           `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time           `gorm:"column:updated_at" json:"updated_at"`
}

func (PenginapanRoom) TableName() string { return "rooms" }

type PenginapanGuest struct {
	ID           int                 `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	RoomID       int                 `gorm:"column:room_id;type:int(11);not null;index" json:"room_id"`
	Room         *PenginapanRoom     `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Name         string              `gorm:"column:name;size:250;not null" json:"name"`
	Nik          *string             `gorm:"column:nik;size:50" json:"nik"`
	Gender       string              `gorm:"column:gender;type:enum('pria','wanita');not null" json:"gender"`
	Harga        int                 `gorm:"column:harga;type:int(11);not null;default:0" json:"harga"`
	CheckinDate  *time.Time          `gorm:"column:checkin_date;type:date" json:"checkin_date"`
	CheckinTime  *string             `gorm:"column:checkin_time;type:time(0)" json:"checkin_time"`
	CheckoutDate *time.Time          `gorm:"column:checkout_date;type:date" json:"checkout_date"`
	CheckoutTime *string             `gorm:"column:checkout_time;type:time(0)" json:"checkout_time"`
	UserID       *uint               `gorm:"column:user_id;type:int(11)" json:"user_id"`
	CreatedAt    time.Time           `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time           `gorm:"column:updated_at" json:"updated_at"`
	TotalBayar   *int                `gorm:"-" json:"total_bayar,omitempty"`
	Sisa         *int                `gorm:"-" json:"sisa,omitempty"`
	Lunas        *bool               `gorm:"-" json:"lunas,omitempty"`
}

func (PenginapanGuest) TableName() string { return "guests" }

type Payment struct {
	ID              int             `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	Amount          int             `gorm:"column:amount;type:int(11);not null;default:0" json:"amount"`
	PaymentMethodID int             `gorm:"column:payment_method_id;type:int(11);not null" json:"payment_method_id"`
	PaymentMethod   *PaymentMethod  `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
	SourceID        int             `gorm:"column:source_id;type:int(11);not null" json:"source_id"`
	Source          *PurchaseSource `gorm:"foreignKey:SourceID" json:"source,omitempty"`
	Status          string          `gorm:"column:status;type:enum('pending','lunas','batal');not null;default:pending" json:"status"`
	Note            *string         `gorm:"column:note;size:250" json:"note"`
	PaidAt          *time.Time      `gorm:"column:paid_at" json:"paid_at"`
	UserID          *uint           `gorm:"column:user_id;type:int(11)" json:"user_id"`
	CreatedAt       time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at" json:"updated_at"`
	Guests          []PaymentGuest  `gorm:"foreignKey:PaymentID" json:"guests,omitempty"`
}

func (Payment) TableName() string { return "payments" }

type PaymentGuest struct {
	ID        int              `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	PaymentID int              `gorm:"column:payment_id;type:int(11);not null;uniqueIndex:idx_payment_guest" json:"payment_id"`
	GuestID   int              `gorm:"column:guest_id;type:int(11);not null;uniqueIndex:idx_payment_guest" json:"guest_id"`
	Amount    int              `gorm:"column:amount;type:int(11);not null;default:0" json:"amount"`
	Guest     *PenginapanGuest `gorm:"foreignKey:GuestID" json:"guest,omitempty"`
}

func (PaymentGuest) TableName() string { return "payment_guests" }

type RoomHistory struct {
	ID         int       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	GuestID    int       `gorm:"column:guest_id;type:int(11);not null;index" json:"guest_id"`
	RoomID     int       `gorm:"column:room_id;type:int(11);not null;index" json:"room_id"`
	Status     string    `gorm:"column:status;type:enum('check-in','check-out','pindah');not null" json:"status"`
	OccurredAt time.Time `gorm:"column:occurred_at" json:"occurred_at"`
	UserID     *uint     `gorm:"column:user_id;type:int(11)" json:"user_id"`
	Note       *string   `gorm:"column:note;size:250" json:"note"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (RoomHistory) TableName() string { return "room_histories" }

type RoomLayout struct {
	ID         int                 `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	LocationID *int                `gorm:"column:location_id;type:int(11);index" json:"location_id"`
	Location   *PenginapanLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	RoomID     *int                `gorm:"column:room_id;type:int(11);index" json:"room_id"`
	Room       *PenginapanRoom     `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	RowNumber  *int                `gorm:"column:row_number;type:int(11)" json:"row_number"`
	Urutan     int                 `gorm:"column:urutan;type:int(11);not null;default:0" json:"urutan"`
	CreatedAt  time.Time           `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time           `gorm:"column:updated_at" json:"updated_at"`
}

func (RoomLayout) TableName() string { return "room_layouts" }
