package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidAuthID = errors.New("auth id must be a valid ULID")
	ulidPattern      = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
)

func ValidateAuthID(value string) error {
	if !ulidPattern.MatchString(strings.TrimSpace(value)) {
		return ErrInvalidAuthID
	}

	return nil
}
