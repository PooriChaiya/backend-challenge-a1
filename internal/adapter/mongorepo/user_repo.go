// Package mongorepo implements port.UserRepository against MongoDB.
package mongorepo

import (
	"context"
	"errors"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type userDoc struct {
	ID           string    `bson:"_id"`
	Name         string    `bson:"name"`
	Email        string    `bson:"email"`
	PasswordHash string    `bson:"password_hash"`
	CreatedAt    bson.DateTime `bson:"created_at"`
}

func toDoc(u *domain.User) userDoc {
	return userDoc{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    bson.NewDateTimeFromTime(u.CreatedAt),
	}
}

func fromDoc(d userDoc) domain.User {
	return domain.User{
		ID:           d.ID,
		Name:         d.Name,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		CreatedAt:    d.CreatedAt.Time(),
	}
}

type Repo struct {
	col *mongo.Collection
}

// New builds the repo and ensures the unique email index exists.
func New(ctx context.Context, db *mongo.Database) (*Repo, error) {
	col := db.Collection("users")
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_email"),
	})
	if err != nil {
		return nil, err
	}
	return &Repo{col: col}, nil
}

func (r *Repo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.col.InsertOne(ctx, toDoc(u))
	if mongo.IsDuplicateKeyError(err) {
		return domain.ErrDuplicateEmail
	}
	return err
}

func (r *Repo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

func (r *Repo) findOne(ctx context.Context, filter bson.M) (*domain.User, error) {
	var d userDoc
	err := r.col.FindOne(ctx, filter).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u := fromDoc(d)
	return &u, nil
}

func (r *Repo) List(ctx context.Context) ([]domain.User, error) {
	cur, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]domain.User, len(docs))
	for i, d := range docs {
		out[i] = fromDoc(d)
	}
	return out, nil
}

func (r *Repo) Update(ctx context.Context, u *domain.User) error {
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": u.ID}, bson.M{"$set": bson.M{
		"name":  u.Name,
		"email": u.Email,
	}})
	if mongo.IsDuplicateKeyError(err) {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{})
}
