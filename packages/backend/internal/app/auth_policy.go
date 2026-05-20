package app

import (
	"crypto/rand"
	"time"

	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/id"
)

func newAuthIDPolicy() id.AuthIDPolicy {
	return id.AuthIDPolicy{
		New: func() string {
			ulid, err := id.NewULID(time.Now().UTC(), rand.Reader)
			if err != nil {
				panic(err)
			}

			return ulid
		},
		Validate: domain.ValidateAuthID,
	}
}
