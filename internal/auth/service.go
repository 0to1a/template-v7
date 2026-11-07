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

// Unknown users and bad codes look the same
var errUnauthenticated = gen.Error(gen.Unauthenticated, "invalid email or code")

// Bad token and unknown account look the same
var errGoogleUnauthenticated = gen.Error(gen.Unauthenticated, "google sign-in failed")

type Service struct {
	repo                Repository
	delivery            LoginCodeSender
	jwtManager          *JWTManager
	isGuestRegistration bool

	// nil = Google login disabled
	firebase *gen.FirebaseConfig
	verifier IDTokenVerifier
}

// Login codes are derived from jwtManager's secret and clock
func NewService(repo Repository, delivery LoginCodeSender, jwtManager *JWTManager, isGuestRegistration bool) *Service {
	return &Service{
		repo:                repo,
		delivery:            delivery,
		jwtManager:          jwtManager,
		isGuestRegistration: isGuestRegistration,
	}
}

// Enables Google login; cfg is served to the SPA as-is
func (s *Service) WithGoogle(cfg gen.FirebaseConfig, verifier IDTokenVerifier) {
	s.firebase = &cfg
	s.verifier = verifier
}

// Never reveals whether the account exists; always 204
func (s *Service) RequestLogin(ctx context.Context, req *gen.RequestLoginRequest) error {
	normalized := normalizeEmail(req.Email)
	user, err := s.findOrRegister(ctx, normalized)
	if errors.Is(err, ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	code := generateCode(s.jwtManager.secret, user.PublicUUID, normalized, s.jwtManager.now())
	// Failure not surfaced: it would only happen for existing accounts
	_ = s.delivery.SendLoginCode(ctx, user.Email, code)
	return nil
}

func (s *Service) SubmitLogin(ctx context.Context, req *gen.SubmitLoginRequest) (*gen.Tokens, error) {
	normalized := normalizeEmail(req.Email)
	user, err := s.repo.GetActiveUserByEmail(ctx, normalized)
	if errors.Is(err, ErrUserNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}

	if !verifyCode(s.jwtManager.secret, user.PublicUUID, normalized, req.Code, s.jwtManager.now()) {
		return nil, errUnauthenticated
	}
	return s.issueTokens(ctx, user.PublicUUID)
}

func (s *Service) GetAuthConfig(context.Context, *gen.GetAuthConfigRequest) (*gen.GetAuthConfigResponse, error) {
	return &gen.GetAuthConfigResponse{Firebase: s.firebase}, nil
}

func (s *Service) GoogleLogin(ctx context.Context, req *gen.GoogleLoginRequest) (*gen.Tokens, error) {
	if s.firebase == nil {
		return nil, gen.Error(gen.FailedPrecondition, "google login is not configured")
	}

	email, err := s.verifier.VerifyIDToken(ctx, req.IDToken)
	if err != nil {
		return nil, errGoogleUnauthenticated
	}
	user, err := s.findOrRegister(ctx, normalizeEmail(email))
	if errors.Is(err, ErrUserNotFound) {
		return nil, errGoogleUnauthenticated
	}
	if err != nil {
		return nil, err
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

// Guest registration decides whether an unknown email gets an account
func (s *Service) findOrRegister(ctx context.Context, normalizedEmail string) (User, error) {
	user, err := s.repo.GetActiveUserByEmail(ctx, normalizedEmail)
	if errors.Is(err, ErrUserNotFound) && s.isGuestRegistration {
		return s.repo.CreateUser(ctx, normalizedEmail)
	}
	return user, err
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
	expiresAt := s.jwtManager.now().Add(refreshTokenTTL)
	if err := s.repo.CreateRefreshToken(ctx, publicUUID, hashRefreshToken(refreshToken), expiresAt); err != nil {
		return nil, err
	}
	return &gen.Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Only the hash is stored, so a database leak yields no usable tokens
func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
