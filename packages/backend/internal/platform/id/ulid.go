package id

import (
	"fmt"
	"io"
	"time"

	ulid "github.com/oklog/ulid/v2"
)

func NewULID(now time.Time, entropy io.Reader) (string, error) {
	value, err := ulid.New(ulid.Timestamp(now.UTC()), entropy)
	if err != nil {
		return "", fmt.Errorf("read ulid entropy: %w", err)
	}

	return value.String(), nil
}
