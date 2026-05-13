package id

import "errors"

// ErrInvalidIDPolicy は AuthIDPolicy の各フィールドが未設定の場合に返されるエラー。
var ErrInvalidIDPolicy = errors.New("auth id policy is required")

// AuthIDPolicy は認証IDの生成と検証を担当するポリシー。
type AuthIDPolicy struct {
	New      func() string
	Validate func(string) error
}

// Check は指定されたIDをポリシーの検証ルールで検証する。
func (p AuthIDPolicy) Check(id string) error {
	if p.Validate == nil {
		return ErrInvalidIDPolicy
	}

	return p.Validate(id)
}

// Next はポリシーの生成ルールに基づいて新しいIDを発行する。
func (p AuthIDPolicy) Next() (string, error) {
	if p.New == nil {
		return "", ErrInvalidIDPolicy
	}

	return p.New(), nil
}
