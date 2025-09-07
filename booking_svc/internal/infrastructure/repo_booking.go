package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/souravlahiri/booking_svc/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BookingRepo struct {
	col *mongo.Collection
}

func NewBookingRepo(db *mongo.Database) *BookingRepo {
	return &BookingRepo{col: db.Collection("bookings")}
}

func (r *BookingRepo) Insert(ctx context.Context, b *domain.Booking) error {
	b.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, b)
	return err
}

func (r *BookingRepo) List(ctx context.Context) ([]domain.Booking, error) {
	cur, err := r.col.Find(ctx, bson.D{}, &options.FindOptions{
		Sort: bson.D{{"created_at", -1}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Booking
	for cur.Next(ctx) {
		var b domain.Booking
		if err := cur.Decode(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// Update booking upon accept event (idempotent)
func (r *BookingRepo) AcceptBooking(ctx context.Context, bookingID, driverID string) error {
	// Only update if ride_status != "Accepted"
	filter := bson.M{"booking_id": bookingID, "ride_status": bson.M{"$ne": "Accepted"}}
	update := bson.M{
		"$set": bson.M{
			"ride_status": "Accepted",
			"driver_id":   driverID,
		},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		// Could be already accepted or not found
		return errors.New("not found or already accepted")
	}
	return nil
}
