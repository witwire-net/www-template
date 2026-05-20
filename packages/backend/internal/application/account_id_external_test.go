package application_test

import domain "www-template/packages/backend/internal/domain"

func testAccountID(raw string) domain.AccountID {
	accountID, err := domain.NewAccountID(raw)
	if err != nil {
		panic(err)
	}
	return accountID
}
