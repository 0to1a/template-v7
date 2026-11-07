package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"project/internal/gen/db"
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
	var id pgtype.UUID
	if err := id.Scan(publicUUID); err != nil {
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
	var id pgtype.UUID
	if err := id.Scan(publicUUID); err != nil {
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
