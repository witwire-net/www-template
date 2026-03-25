package types

import "errors"

var ErrInvalidIDPolicy = errors.New("auth id policy is required")

type AuthIDPolicy struct {
	New      func() string
	Validate func(string) error
}

func (p AuthIDPolicy) Check(id string) error {
	if p.Validate == nil {
		return ErrInvalidIDPolicy
	}

	return p.Validate(id)
}

func (p AuthIDPolicy) Next() (string, error) {
	if p.New == nil {
		return "", ErrInvalidIDPolicy
	}

	return p.New(), nil
}
