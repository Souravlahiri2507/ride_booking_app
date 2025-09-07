package infrastructure

import (
	"context"

	"github.com/souravlahiri/driver_svc/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DriverRepo struct {
	col *mongo.Collection
}

func NewDriverRepo(db *mongo.Database) *DriverRepo {
	return &DriverRepo{col: db.Collection("drivers")}
}

func (r *DriverRepo) SeedIfEmpty(ctx context.Context) error {
	count, err := r.col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if count == 0 {
		docs := []interface{}{
			domain.Driver{ID: "d-1", Name: "Asha", Available: true},
			domain.Driver{ID: "d-2", Name: "Ravi", Available: true},
		}
		_, err := r.col.InsertMany(ctx, docs)
		return err
	}
	return nil
}

func (r *DriverRepo) List(ctx context.Context) ([]domain.Driver, error) {
	cur, err := r.col.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.Driver
	for cur.Next(ctx) {
		var d domain.Driver
		_ = cur.Decode(&d)
		out = append(out, d)
	}
	return out, nil
}
