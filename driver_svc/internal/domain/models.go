package domain

import "time"

type Driver struct {
	ID        string `json:"id" bson:"id"`
	Name      string `json:"name" bson:"name"`
	Available bool   `json:"available" bson:"available"`
}

type Location struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}

type Job struct {
	BookingID string    `json:"booking_id" bson:"booking_id"`
	PickUp    Location  `json:"pickuploc" bson:"pickuploc"`
	DropOff   Location  `json:"dropoff" bson:"dropoff"`
	Price     float64   `json:"price" bson:"price"`
	Taken     bool      `json:"taken" bson:"taken"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
