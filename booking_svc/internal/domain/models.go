package domain

import "time"

type Location struct {
	Lat float64 `json:"lat" bson:"lat"`
	Lng float64 `json:"lng" bson:"lng"`
}

type Booking struct {
	BookingID  string    `json:"booking_id" bson:"booking_id"`
	PickUp     Location  `json:"pickuploc" bson:"pickuploc"`
	DropOff    Location  `json:"dropoff" bson:"dropoff"`
	Price      float64   `json:"price" bson:"price"`
	RideStatus string    `json:"ride_status" bson:"ride_status"` // Requested or Accepted
	DriverID   *string   `json:"driver_id,omitempty" bson:"driver_id,omitempty"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
}
