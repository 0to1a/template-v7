package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	gen "project/internal/gen/api"
)

const refreshTokenTTL = 30 * 24 * time.Hour

// Unknown, expired, revoked and reused tokens look the same
var errRefreshUnauthenticated = gen.Error(gen.Unauthenticated, "invalid refresh token")

// Same generic error for unknown users and bad codes
var errUnauthenticated = gen.Error(gen.Unauthenticated, "invalid email or code")

// Bad token and unknown account look the same
var errGoogleUnauthenticated = gen.Error(gen.Unauthenticated, "google sign-in failed")

type Service struct {
	repo                Repository
	delivery            LoginCodeSender
	jwtManager          *JWTManager
	now                 func() time.Time
	isGuestRegistration bool

	// nil = Google login disabled
	firebase *gen.FirebaseConfig
	verifier IDTokenVerifier
}

// TOTP secret derived from jwtManager's signing secret
func NewService(repo Repository, delivery LoginCodeSender, jwtManager *JWTManager, isGuestRegistration bool) *Service {
	return &Service{
		repo:                repo,
		delivery:            delivery,
		jwtManager:          jwtManager,
		now:                 time.Now,
		isGuestRegistration: isGuestRegistration,
	}
}

// Never reveals whether the account already existed; always 204
func (s *Service) RequestLogin(ctx context.Context, req *gen.RequestLoginRequest) error {
	normalized := normalizeEmail(req.Email)

	user, err := s.repo.GetActiveUserByEmail(ctx, normalized)
	if errors.Is(err, ErrUserNotFound) {
		if !s.isGuestRegistration {
			return nil
		}
		user, err = s.repo.CreateUser(ctx, normalized)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	code := s.jwtManager.generateLoginCode(user.PublicUUID, normalized, s.now())
	// Failure not surfaced: must not leak account existence
	_ = s.delivery.SendLoginCode(ctx, user.Email, code)
	return nil
}

// Unknown users and bad codes return the same error
func (s *Service) SubmitLogin(ctx context.Context, req *gen.SubmitLoginRequest) (*gen.Tokens, error) {
	normalized := normalizeEmail(req.Email)

	user, err := s.repo.GetActiveUserByEmail(ctx, normalized)
	if errors.Is(err, ErrUserNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}

	if !s.jwtManager.verifyLoginCode(user.PublicUUID, normalized, req.Code, s.now()) {
		return nil, errUnauthenticated
	}

	return s.issueTokens(ctx, user.PublicUUID)
}

// Rotates: the presented token is spent, a new pair is returned
func (s *Service) Refresh(ctx context.Context, req *gen.RefreshRequest) (*gen.Tokens, error) {
	hash := hashRefreshToken(req.RefreshToken)
	publicUUID, err := s.repo.RevokeRefreshToken(ctx, hash)
	if errors.Is(err, ErrRefreshTokenNotFound) {
		if err := s.repo.RevokeRefreshTokensIfReused(ctx, hash); err != nil {
			return nil, err
		}
		return nil, errRefreshUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, publicUUID)
}

// Public so an expired access token can still log out; the refresh token is the credential
func (s *Service) Logout(ctx context.Context, req *gen.LogoutRequest) error {
	_, err := s.repo.RevokeRefreshToken(ctx, hashRefreshToken(req.RefreshToken))
	if errors.Is(err, ErrRefreshTokenNotFound) {
		return nil
	}
	return err
}

func (s *Service) issueTokens(ctx context.Context, publicUUID string) (*gen.Tokens, error) {
	accessToken, err := s.jwtManager.Issue(publicUUID)
	if err != nil {
		return nil, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(raw)
	if err := s.repo.CreateRefreshToken(ctx, publicUUID, hashRefreshToken(refreshToken), s.now().Add(refreshTokenTTL)); err != nil {
		return nil, err
	}
	return &gen.Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Only the hash is stored, so a database leak yields no usable tokens
func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Enables Google login; cfg is served to the SPA as-is
func (s *Service) WithGoogle(cfg gen.FirebaseConfig, verifier IDTokenVerifier) *Service {
	s.firebase = &cfg
	s.verifier = verifier
	return s
}

func (s *Service) GetAuthConfig(context.Context, *gen.GetAuthConfigRequest) (*gen.GetAuthConfigResponse, error) {
	return &gen.GetAuthConfigResponse{Firebase: s.firebase}, nil
}

// Same account rules as email login: guest registration decides sign-up
func (s *Service) GoogleLogin(ctx context.Context, req *gen.GoogleLoginRequest) (*gen.Tokens, error) {
	if s.firebase == nil {
		return nil, gen.Error(gen.FailedPrecondition, "google login is not configured")
	}

	email, err := s.verifier.VerifyIDToken(ctx, req.IDToken)
	if err != nil {
		return nil, errGoogleUnauthenticated
	}
	normalized := normalizeEmail(email)

	user, err := s.repo.GetActiveUserByEmail(ctx, normalized)
	if errors.Is(err, ErrUserNotFound) {
		if !s.isGuestRegistration {
			return nil, errGoogleUnauthenticated
		}
		user, err = s.repo.CreateUser(ctx, normalized)
	}
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user.PublicUUID)
}
