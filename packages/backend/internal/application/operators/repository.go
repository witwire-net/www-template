package operators

import (
	"context"
	"time"
)

// ─── Repository port ───────────────────────────────────────────────────────

// OperatorRepository は Operator setup / creation が必要とする永続化 port である。
//
// 役割:
//   - application 層から GORM や SQL を隠し、operator root と passkey credential の transaction を adapter に閉じる。
//   - setup token は opaque hash と expiry だけを保存し、平文 token は repository 境界へ渡さない。
//   - passkey 登録完了時は token 消費、passkey 保存、registration state 更新を同一 transaction で実行する。
type OperatorRepository interface {
	// CountOperators は現在登録済みの operator 件数を返す。
	// bootstrap gate 判定に使い、保存層障害時は error を返す。
	CountOperators(ctx context.Context) (int64, error)
	// CreateInitialOperatorWithPasskey は初回 operator と passkey を同一 transaction で保存する。
	// 同時初回作成は ErrOperatorConflict として扱う。
	CreateInitialOperatorWithPasskey(ctx context.Context, record InitialOperatorRecord) (OperatorRecord, error)
	// CreateOperatorWithSetupToken は追加 operator と setup token hash を同一 transaction で保存する。
	// email 重複は ErrOperatorConflict として返す。domain rule 違反は ErrOperatorInvalidInput として返す。
	CreateOperatorWithSetupToken(ctx context.Context, record OperatorCreationRecord) (OperatorRecord, error)
	// DeletePendingOperatorSetup は delivery 失敗時の pending operator を削除する。
	// 削除対象が存在しない場合は error を返さない。
	DeletePendingOperatorSetup(ctx context.Context, operatorID string) error
	// FindOperatorBySetupToken は setup token hash と一致する operator を検索する。
	// match callback は保存済み hash を受け取り、平文 token は repository 境界へ渡さない。
	FindOperatorBySetupToken(ctx context.Context, now time.Time, match func(hash string) bool) (SetupRecord, error)
	// CompleteOperatorSetupWithPasskey は setup token 消費と passkey 保存を同一 transaction で実行する。
	// 既消費 token は ErrOperatorForbidden として扱う。
	CompleteOperatorSetupWithPasskey(ctx context.Context, record SetupCompletionRecord) (OperatorRecord, error)
}

// ─── Record DTO ────────────────────────────────────────────────────────────

// OperatorRecord は operator 永続化後の primitive snapshot である。
//
// 役割:
//   - HTTP response や後続 use case が必要とする primitive だけを保持し、GORM record や DB column tag を application 境界へ出さない。
//   - Email、Role は domain object が検証した canonical 文字列であり、repository は値を再解釈しない。
type OperatorRecord struct {
	OperatorID               string
	Email                    string
	Role                     string
	Active                   bool
	PasskeyRegistrationState string
	CreatedAt                time.Time
}

// SetupRecord は setup token に一致した operator の primitive snapshot である。
//
// 役割:
//   - setup flow 開始時に repository が返す最小限の operator 情報であり、passkey credential は含めない。
//   - OperatorID は challenge 発行と setup 完了の両方で使用する識別子である。
type SetupRecord struct {
	OperatorID string
	Email      string
	Role       string
	Active     bool
}

// PasskeyRecord は検証済み WebAuthn credential の保存 DTO である。
//
// 役割:
//   - WebAuthn provider が検証した credential data を repository transaction へ渡すための application 境界型である。
//   - credential handle、public key、sign count など認証検証用の内部状態を保持する。
//   - 平文 secret は含めず、credential ID は operator credential 専用 ULID として発行する。
type PasskeyRecord struct {
	CredentialID     string
	CredentialHandle string
	PublicKey        []byte
	SignCount        uint32
	AAGUID           []byte
	BackupEligible   bool
	BackupState      bool
	Transports       []string
}

// InitialOperatorRecord は初回 operator と passkey を同一 transaction で保存する DTO である。
//
// 役割:
//   - bootstrap flow で作成される初回 operator の operator root と passkey credential をまとめて repository へ渡す。
//   - OperatorID は requestID と同じ ULID に固定し、WebAuthn session の user handle と DB 作成 ID を一致させる。
type InitialOperatorRecord struct {
	OperatorID  string
	Email       string
	Passkey     PasskeyRecord
	CompletedAt time.Time
}

// OperatorCreationRecord は追加 operator と setup token hash を保存する DTO である。
//
// 役割:
//   - acting operator が追加 operator を作成する際に、operator root と setup token の opaque hash/expiry を repository へ渡す。
//   - 平文 setup token は含めず、hash だけを保存する。
type OperatorCreationRecord struct {
	OperatorID          string
	Email               string
	Role                string
	SetupTokenHash      string
	SetupTokenExpiresAt time.Time
	CreatedAt           time.Time
}

// SetupCompletionRecord は setup token 消費と passkey 保存を同一 transaction で実行する DTO である。
//
// 役割:
//   - setup flow 完了時に、operator ID、token 照合 callback、passkey credential を repository へ渡す。
//   - SetupTokenMatches は保存済み hash を受け取り、平文 token を repository 境界へ渡さない設計を維持する。
type SetupCompletionRecord struct {
	OperatorID        string
	SetupTokenMatches func(hash string) bool
	Passkey           PasskeyRecord
	CompletedAt       time.Time
}
