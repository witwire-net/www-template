package persistence

import (
	"context"
	"sort"
	"sync"
	"time"

	"witwire.net/www-template/packages/backend/internal/domain"
)

type MemoryProfileRepository struct {
	mu       sync.RWMutex
	nextID   int64
	profiles map[int64]domain.Profile
}

func NewMemoryProfileRepository() *MemoryProfileRepository {
	return &MemoryProfileRepository{
		nextID:   1,
		profiles: map[int64]domain.Profile{},
	}
}

func (r *MemoryProfileRepository) List(context.Context) ([]domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles := make([]domain.Profile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		profiles = append(profiles, profile)
	}

	sort.Slice(profiles, func(i int, j int) bool {
		return profiles[i].ID() < profiles[j].ID()
	})

	return profiles, nil
}

func (r *MemoryProfileRepository) GetByID(_ context.Context, id int64) (domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.profiles[id]
	if !ok {
		var empty domain.Profile
		return empty, domain.ErrProfileNotFound
	}

	return profile, nil
}

func (r *MemoryProfileRepository) Create(_ context.Context, input domain.CreateProfileInput) (domain.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	profile, err := domain.NewProfile(r.nextID, time.Now().UTC(), input)
	if err != nil {
		var empty domain.Profile
		return empty, err
	}

	r.profiles[profile.ID()] = profile
	r.nextID++

	return profile, nil
}
