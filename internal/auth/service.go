package auth

import (
	"context"
	"errors"
	"time"

	gen "project/internal/gen/api"
)

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
func (s *Service) SubmitLogin(ctx context.Context, req *gen.SubmitLoginRequest) (*gen.SubmitLoginResponse, error) {
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

	token, err := s.jwtManager.Issue(user.PublicUUID)
	if err != nil {
		return nil, err
	}
	return &gen.SubmitLoginResponse{AccessToken: token}, nil
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
func (s *Service) GoogleLogin(ctx context.Context, req *gen.GoogleLoginRequest) (*gen.GoogleLoginResponse, error) {
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

	token, err := s.jwtManager.Issue(user.PublicUUID)
	if err != nil {
		return nil, err
	}
	return &gen.GoogleLoginResponse{AccessToken: token}, nil
}
