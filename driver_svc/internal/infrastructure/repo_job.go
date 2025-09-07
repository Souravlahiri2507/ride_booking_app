package infrastructure

import (
	"context"
	"time"

	"github.com/souravlahiri/driver_svc/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JobRepo struct {
	col *mongo.Collection
}

func NewJobRepo(db *mongo.Database) *JobRepo {
	return &JobRepo{col: db.Collection("jobs")}
}

func (r *JobRepo) InsertJob(ctx context.Context, j *domain.Job) error {
	j.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, j)
	return err
}

func (r *JobRepo) ListOpenJobs(ctx context.Context) ([]domain.Job, error) {
	cur, err := r.col.Find(ctx, bson.M{"taken": false}, &options.FindOptions{Sort: bson.D{{"created_at", -1}}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Job
	for cur.Next(ctx) {
		var j domain.Job
		_ = cur.Decode(&j)
		out = append(out, j)
	}
	return out, nil
}

// Atomically claim job: ensure taken==false then set taken=true
func (r *JobRepo) ClaimJob(ctx context.Context, bookingID, driverID string) (bool, error) {
	filter := bson.M{"booking_id": bookingID, "taken": false}
	update := bson.M{"$set": bson.M{"taken": true}}
	res := r.col.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.Before))
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, res.Err()
	}
	return true, nil
}
