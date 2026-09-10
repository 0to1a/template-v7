package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"project/internal/gen/db"
)

// ErrUserNotFound means the lookup found no active user.
var ErrUserNotFound = errors.New("auth: user not found")

// Decoupled from generated sqlc row type
type User struct {
	PublicUUID string
	Email      string
}

// Lets tests substitute a fake, no real database needed
type Repository interface {
	GetActiveUserByEmail(ctx context.Context, normalizedEmail string) (User, error)
	CreateUser(ctx context.Context, normalizedEmail string) (User, error)
}

type repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) Repository {
	return &repository{queries: queries}
}

func (r *repository) GetActiveUserByEmail(ctx context.Context, normalizedEmail string) (User, error) {
	row, err := r.queries.GetActiveUserByEmail(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return User{
		PublicUUID: row.PublicUuid.String(),
		Email:      row.Email,
	}, nil
}

func (r *repository) CreateUser(ctx context.Context, normalizedEmail string) (User, error) {
	row, err := r.queries.CreateUser(ctx, normalizedEmail)
	if err != nil {
		return User{}, err
	}
	return User{
		PublicUUID: row.PublicUuid.String(),
		Email:      row.Email,
	}, nil
}
