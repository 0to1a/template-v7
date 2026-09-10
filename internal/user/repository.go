package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"project/internal/gen/db"
	"project/internal/platform/database"
)

// ErrUserNotFound means the lookup found no active user.
var ErrUserNotFound = errors.New("user: user not found")

// Decoupled from generated sqlc row type
type Profile struct {
	Email       string
	DisplayName string
}

// Lets tests substitute a fake, no real database needed
type Repository interface {
	GetProfileByPublicUUID(ctx context.Context, publicUUID string) (Profile, error)
	UpdateDisplayName(ctx context.Context, publicUUID, displayName string) (Profile, error)
}

type repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) Repository {
	return &repository{queries: queries}
}

func (r *repository) GetProfileByPublicUUID(ctx context.Context, publicUUID string) (Profile, error) {
	id, err := database.ParsePublicUUID(publicUUID)
	if err != nil {
		return Profile{}, ErrUserNotFound
	}

	row, err := r.queries.GetUserByPublicUUID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrUserNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return Profile{Email: row.Email, DisplayName: row.DisplayName}, nil
}

func (r *repository) UpdateDisplayName(ctx context.Context, publicUUID, displayName string) (Profile, error) {
	id, err := database.ParsePublicUUID(publicUUID)
	if err != nil {
		return Profile{}, ErrUserNotFound
	}

	row, err := r.queries.UpdateDisplayName(ctx, db.UpdateDisplayNameParams{
		PublicUuid:  id,
		DisplayName: displayName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrUserNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return Profile{Email: row.Email, DisplayName: row.DisplayName}, nil
}
