package user

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	gen "project/internal/gen/api"
)

// Empty string is allowed: means no display name set
const maxDisplayNameLength = 100

// Acting user always comes from req.Auth (set by the auth middleware),
// never from a client-supplied identifier.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, req *gen.GetProfileRequest) (*gen.Profile, error) {
	profile, err := s.repo.GetProfileByPublicUUID(ctx, req.Auth.ID)
	if errors.Is(err, ErrUserNotFound) {
		return nil, gen.Error(gen.NotFound, "profile not found")
	}
	if err != nil {
		return nil, err
	}
	return toGen(profile), nil
}

// Empty trimmed value clears the name; over max is rejected
func (s *Service) UpdateProfile(ctx context.Context, req *gen.UpdateProfileRequest) (*gen.Profile, error) {
	var name string
	if req.DisplayName != nil {
		name = *req.DisplayName
	}
	trimmed := strings.TrimSpace(name)
	if utf8.RuneCountInString(trimmed) > maxDisplayNameLength {
		return nil, gen.Errorf(gen.InvalidArgument, "display_name must be at most %d characters", maxDisplayNameLength)
	}
	profile, err := s.repo.UpdateDisplayName(ctx, req.Auth.ID, trimmed)
	if errors.Is(err, ErrUserNotFound) {
		return nil, gen.Error(gen.NotFound, "profile not found")
	}
	if err != nil {
		return nil, err
	}
	return toGen(profile), nil
}

func toGen(p Profile) *gen.Profile {
	return &gen.Profile{Email: p.Email, DisplayName: p.DisplayName}
}
