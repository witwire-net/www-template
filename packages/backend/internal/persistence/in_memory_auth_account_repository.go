package persistence

import (
	"context"
	"sync"

	"www-template/packages/backend/internal/domain"
)

type InMemoryAuthAccountRepository struct {
	mu           sync.RWMutex
	byAccountID  map[string]domain.AuthAccount
	byIdentifier map[string]string
	byEmail      map[string]string
	byCredential map[string]string
}

func NewInMemoryAuthAccountRepository() *InMemoryAuthAccountRepository {
	account, _ := domain.NewAuthAccount(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"member@example.com",
		"member@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FB0",
		"existing-credential",
	)

	repo := &InMemoryAuthAccountRepository{
		byAccountID:  map[string]domain.AuthAccount{},
		byIdentifier: map[string]string{},
		byEmail:      map[string]string{},
		byCredential: map[string]string{},
	}
	repo.store(account)
	return repo
}

func (r *InMemoryAuthAccountRepository) FindByIdentifier(_ context.Context, identifier string) (domain.AuthAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findByLookup(r.byIdentifier, identifier)
}

func (r *InMemoryAuthAccountRepository) FindByCredential(_ context.Context, credential string) (domain.AuthAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findByLookup(r.byCredential, credential)
}

func (r *InMemoryAuthAccountRepository) FindByEmail(_ context.Context, email string) (domain.AuthAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findByLookup(r.byEmail, email)
}

func (r *InMemoryAuthAccountRepository) ReplacePasskey(_ context.Context, accountID string, passkeyCredentialID string, credential string) (domain.AuthAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.byAccountID[accountID]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	updated, err := domain.NewAuthAccount(account.AccountID(), account.Identifier(), account.Email(), passkeyCredentialID, credential)
	if err != nil {
		return emptyAuthAccount(), err
	}
	delete(r.byCredential, account.CredentialHandle())
	r.store(updated)
	return updated, nil
}

func (r *InMemoryAuthAccountRepository) findByLookup(lookup map[string]string, key string) (domain.AuthAccount, error) {
	accountID, ok := lookup[key]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	account, ok := r.byAccountID[accountID]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	return account, nil
}

func (r *InMemoryAuthAccountRepository) store(account domain.AuthAccount) {
	r.byAccountID[account.AccountID()] = account
	r.byIdentifier[account.Identifier()] = account.AccountID()
	r.byEmail[account.Email()] = account.AccountID()
	r.byCredential[account.CredentialHandle()] = account.AccountID()
}

func emptyAuthAccount() domain.AuthAccount {
	account, _ := domain.NewAuthAccount(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"placeholder@example.com",
		"placeholder@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"placeholder-credential",
	)
	return account
}
