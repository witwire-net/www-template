package usecases

import (
	"context"
	"errors"
	"time"

	"witwire.net/www-template/packages/backend/internal/domain"
)

var (
	ErrInvalidProfileEmail = errors.New("email is required")
	ErrInvalidProfileName  = errors.New("name is required")
	ErrProfileNotFound     = errors.New("profile not found")
)

type CreateProfileInput struct {
	Email string
	Name  string
}

type Profile struct {
	CreatedAt time.Time
	Email     string
	Name      string
	ID        int64
}

type StatusMessage struct {
	Timestamp time.Time
	Message   string
}

type ProfilesService struct {
	repository domain.ProfileRepository
	clock      func() time.Time
}

func NewProfilesService(repository domain.ProfileRepository, clock func() time.Time) *ProfilesService {
	if clock == nil {
		panic("clock is required")
	}

	return &ProfilesService{
		repository: repository,
		clock:      clock,
	}
}

func (s *ProfilesService) GetStatus(context.Context) StatusMessage {
	return StatusMessage{
		Message:   "Sample backend status ready",
		Timestamp: s.clock(),
	}
}

func (s *ProfilesService) ListProfiles(ctx context.Context) ([]Profile, error) {
	profiles, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	return toProfiles(profiles), nil
}

func (s *ProfilesService) CreateProfile(ctx context.Context, input CreateProfileInput) (Profile, error) {
	domainInput, err := domain.NewCreateProfileInput(input.Email, input.Name)
	if err != nil {
		return Profile{}, mapProfileError(err)
	}

	profile, err := s.repository.Create(ctx, domainInput)
	if err != nil {
		return Profile{}, mapProfileError(err)
	}

	return toProfile(profile), nil
}

func (s *ProfilesService) GetProfile(ctx context.Context, id int64) (Profile, error) {
	profile, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Profile{}, mapProfileError(err)
	}

	return toProfile(profile), nil
}

func mapProfileError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidProfileEmail):
		return ErrInvalidProfileEmail
	case errors.Is(err, domain.ErrInvalidProfileName):
		return ErrInvalidProfileName
	case errors.Is(err, domain.ErrProfileNotFound):
		return ErrProfileNotFound
	default:
		return err
	}
}

func toProfile(profile domain.Profile) Profile {
	return Profile{
		CreatedAt: profile.CreatedAt(),
		Email:     profile.Email(),
		Name:      profile.Name(),
		ID:        profile.ID(),
	}
}

func toProfiles(profiles []domain.Profile) []Profile {
	result := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, toProfile(profile))
	}

	return result
}
