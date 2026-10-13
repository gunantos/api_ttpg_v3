package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type PnP struct {
	Name      string     `bson:"name" json:"name"`
	Gender    string     `bson:"gender" json:"gender"`
	Status    *string    `bson:"status" json:"status"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
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
	DestinationID   int           `bson:"destination_id" json:"destination_id"`
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
