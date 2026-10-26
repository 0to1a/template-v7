package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"project/internal/gen/db"
	"project/internal/platform/database"
)

// ErrUserNotFound means the lookup found no active user.
var ErrUserNotFound = errors.New("auth: user not found")

// ErrRefreshTokenNotFound means the token is unknown, expired, revoked, or its user is gone.
var ErrRefreshTokenNotFound = errors.New("auth: refresh token not found")

// Decoupled from generated sqlc row type
type User struct {
	PublicUUID string
	Email      string
}

// Lets tests substitute a fake, no real database needed
type Repository interface {
	GetActiveUserByEmail(ctx context.Context, normalizedEmail string) (User, error)
	CreateUser(ctx context.Context, normalizedEmail string) (User, error)
	CreateRefreshToken(ctx context.Context, publicUUID string, tokenHash []byte, expiresAt time.Time) error
	// Returns the owner's public UUID; a token can be revoked only once
	RevokeRefreshToken(ctx context.Context, tokenHash []byte) (string, error)
	// No-op unless tokenHash was already revoked; then revokes all of its user's tokens
	RevokeRefreshTokensIfReused(ctx context.Context, tokenHash []byte) error
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

func (r *repository) CreateRefreshToken(ctx context.Context, publicUUID string, tokenHash []byte, expiresAt time.Time) error {
	id, err := database.ParsePublicUUID(publicUUID)
	if err != nil {
		return err
	}
	return r.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		TokenHash:  tokenHash,
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
		PublicUuid: id,
	})
}

func (r *repository) RevokeRefreshToken(ctx context.Context, tokenHash []byte) (string, error) {
	publicUUID, err := r.queries.RevokeRefreshToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRefreshTokenNotFound
	}
	if err != nil {
		return "", err
	}
	return publicUUID.String(), nil
}

func (r *repository) RevokeRefreshTokensIfReused(ctx context.Context, tokenHash []byte) error {
	return r.queries.RevokeRefreshTokensIfReused(ctx, tokenHash)
}
