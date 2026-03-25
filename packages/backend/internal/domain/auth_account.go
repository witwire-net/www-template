package domain

import "strings"

type AuthAccount struct {
	accountID           string
	identifier          string
	email               string
	passkeyCredentialID string
	credentialHandle    string
}

func NewAuthAccount(accountID string, identifier string, email string, passkeyCredentialID string, credentialHandle string) (AuthAccount, error) {
	if err := ValidateAuthID(accountID); err != nil {
		return AuthAccount{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(passkeyCredentialID); err != nil {
		return AuthAccount{}, ErrInvalidPasskeyCredential
	}
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(credentialHandle) == "" {
		return AuthAccount{}, ErrInvalidChallenge
	}

	return AuthAccount{
		accountID:           accountID,
		identifier:          strings.TrimSpace(identifier),
		email:               strings.TrimSpace(email),
		passkeyCredentialID: passkeyCredentialID,
		credentialHandle:    strings.TrimSpace(credentialHandle),
	}, nil
}

func (a AuthAccount) AccountID() string           { return a.accountID }
func (a AuthAccount) Identifier() string          { return a.identifier }
func (a AuthAccount) Email() string               { return a.email }
func (a AuthAccount) PasskeyCredentialID() string { return a.passkeyCredentialID }
func (a AuthAccount) CredentialHandle() string    { return a.credentialHandle }
