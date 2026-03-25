package app

import (
	"crypto/rand"
	"time"

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/types"
)

func newAuthIDPolicy() types.AuthIDPolicy {
	return types.AuthIDPolicy{
		New: func() string {
			id, err := types.NewULID(time.Now().UTC(), rand.Reader)
			if err != nil {
				panic(err)
			}

			return id
		},
		Validate: domain.ValidateAuthID,
	}
}
