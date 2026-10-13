package request

type PenginapanCheckinGuest struct {
	RoomID      int     `json:"room_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Nik         *string `json:"nik"`
	Gender      string  `json:"gender" binding:"required,oneof=pria wanita"`
	Harga       int     `json:"harga"`
	CheckinDate string  `json:"checkin_date"`
	CheckinTime string  `json:"checkin_time"`
}

type PenginapanAllocation struct {
	GuestIndex int `json:"guest_index"`
	GuestID    int `json:"guest_id"`
	Amount     int `json:"amount"`
}

type PenginapanPaymentInput struct {
	Amount          int                    `json:"amount" binding:"required,min=1"`
	PaymentMethodID int                    `json:"payment_method_id" binding:"required"`
	SourceID        int                    `json:"source_id" binding:"required"`
	Status          string                 `json:"status"`
	Note            *string                `json:"note"`
	Allocations     []PenginapanAllocation `json:"allocations"`
}

type PenginapanCheckinRequest struct {
	Guests  []PenginapanCheckinGuest `json:"guests" binding:"required,min=1,dive"`
	Payment *PenginapanPaymentInput  `json:"payment"`
}

type PenginapanCheckoutRequest struct {
	CheckoutDate string `json:"checkout_date"`
	CheckoutTime string `json:"checkout_time"`
}

type PenginapanPindahRequest struct {
	RoomID int `json:"room_id" binding:"required"`
}

type PenginapanCreatePaymentRequest struct {
	Amount          int                    `json:"amount" binding:"required,min=1"`
	PaymentMethodID int                    `json:"payment_method_id" binding:"required"`
	SourceID        int                    `json:"source_id" binding:"required"`
	Status          string                 `json:"status"`
	Note            *string                `json:"note"`
	GuestIDs        []int                  `json:"guest_ids" binding:"required,min=1"`
	Allocations     []PenginapanAllocation `json:"allocations"`
}

type PenginapanUpdatePaymentRequest struct {
	Amount          *int    `json:"amount"`
	PaymentMethodID *int    `json:"payment_method_id"`
	SourceID        *int    `json:"source_id"`
	Status          *string `json:"status"`
	Note            *string `json:"note"`
}
