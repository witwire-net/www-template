package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidProfileCreatedAt = errors.New("profile created_at is required")
	ErrInvalidProfileEmail     = errors.New("email is required")
	ErrInvalidProfileID        = errors.New("profile id must be greater than zero")
	ErrInvalidProfileName      = errors.New("name is required")
	ErrProfileNotFound         = errors.New("profile not found")
)

type Profile struct {
	createdAt time.Time
	email     string
	name      string
	id        int64
}

type CreateProfileInput struct {
	email string
	name  string
}

type ProfileRepository interface {
	Create(context.Context, CreateProfileInput) (Profile, error)
	GetByID(context.Context, int64) (Profile, error)
	List(context.Context) ([]Profile, error)
}

func NewCreateProfileInput(email string, name string) (CreateProfileInput, error) {
	normalized := CreateProfileInput{
		email: strings.TrimSpace(email),
		name:  strings.TrimSpace(name),
	}

	if normalized.name == "" {
		return CreateProfileInput{}, ErrInvalidProfileName
	}

	if normalized.email == "" {
		return CreateProfileInput{}, ErrInvalidProfileEmail
	}

	return normalized, nil
}

func NewProfile(id int64, createdAt time.Time, input CreateProfileInput) (Profile, error) {
	if id <= 0 {
		return Profile{}, ErrInvalidProfileID
	}

	if createdAt.IsZero() {
		return Profile{}, ErrInvalidProfileCreatedAt
	}

	return Profile{
		createdAt: createdAt,
		email:     input.email,
		name:      input.name,
		id:        id,
	}, nil
}

func ReconstituteProfile(id int64, email string, name string, createdAt time.Time) (Profile, error) {
	input, err := NewCreateProfileInput(email, name)
	if err != nil {
		return Profile{}, err
	}

	return NewProfile(id, createdAt, input)
}

func (i CreateProfileInput) Email() string {
	return i.email
}

func (i CreateProfileInput) Name() string {
	return i.name
}

func (p Profile) CreatedAt() time.Time {
	return p.createdAt
}

func (p Profile) Email() string {
	return p.email
}

func (p Profile) ID() int64 {
	return p.id
}

func (p Profile) Name() string {
	return p.name
}
