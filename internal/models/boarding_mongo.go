package models

import (
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FlexInt menerima BSON int32/int64/double ATAU dokumen lama
// {"_id":…, "id_tujuan":395, …} → diambil id_tujuan-nya.
type FlexInt int

func (f *FlexInt) UnmarshalBSONValue(typ byte, data []byte) error {
	if typ == byte(bson.TypeEmbeddedDocument) {
		var doc struct {
			IDTujuan int `bson:"id_tujuan"`
		}
		if err := bson.Unmarshal(data, &doc); err != nil {
			return err
		}
		*f = FlexInt(doc.IDTujuan)
		return nil
	}
	if typ == byte(bson.TypeNull) {
		*f = 0
		return nil
	}
	if typ == byte(bson.TypeString) {
		var s string
		if err := bson.UnmarshalValue(bson.Type(typ), data, &s); err != nil {
			return err
		}
		if n, err := strconv.Atoi(s); err == nil {
			*f = FlexInt(n)
		} else {
			*f = 0
		}
		return nil
	}
	var v int
	if err := bson.UnmarshalValue(bson.Type(typ), data, &v); err != nil {
		return err
	}
	*f = FlexInt(v)
	return nil
}

type PnP struct {
	Name      string    `bson:"name" json:"name"`
	Gender    string    `bson:"gender" json:"gender"`
	Status    *string   `bson:"status" json:"status"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type BoardingDoc struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	BoardingID      int64         `bson:"boarding_id" json:"boarding_id"`
	TicketID        string        `bson:"ticket_id" json:"ticket_id"`
	DateOfDeparture string        `bson:"date_of_departure" json:"date_of_departure"`
	TimeOfDeparture string        `bson:"time_of_departure" json:"time_of_departure"`
	Nopol           string        `bson:"nopol" json:"nopol"`
	POID            string        `bson:"po_id" json:"po_id"`
	PO              bson.M        `bson:"po,omitempty" json:"po,omitempty"`
	DestinationID   FlexInt       `bson:"destination_id" json:"destination_id"`
	Destination     bson.M        `bson:"destination,omitempty" json:"destination,omitempty"`
	PNP             []PnP         `bson:"pnp" json:"pnp"`
	UserID          uint          `bson:"user_id" json:"user_id"`
	User            bson.M        `bson:"user,omitempty" json:"user,omitempty"`
	Status          string        `bson:"status" json:"status"`
	NIK             string        `bson:"nik,omitempty" json:"nik,omitempty"`
	Address         string        `bson:"address,omitempty" json:"address,omitempty"`
	City            string        `bson:"city,omitempty" json:"city,omitempty"`
	Province        string        `bson:"province,omitempty" json:"province,omitempty"`
	Telp            string        `bson:"telp,omitempty" json:"telp,omitempty"`
	Email           string        `bson:"email,omitempty" json:"email,omitempty"`
	CreatedAt       time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time     `bson:"updatedAt" json:"updatedAt"`
}
