package operators

import (
	"context"
	"time"

	authapplication "www-template/packages/backend/internal/application/auth"
)

// ─── Port interface ────────────────────────────────────────────────────────

// SetupTokenDeliveryPort は setup token 平文を backend-owned secure channel で配送する port である。
//
// 役割:
//   - operator creation response へ setup token 平文を返さないため、配送副作用を application 境界で抽象化する。
//   - 実装は SMTP などの secure delivery を使い、ログや error へ token 平文を含めてはならない。
type SetupTokenDeliveryPort interface {
	SendOperatorSetupToken(ctx context.Context, delivery SetupTokenDelivery) error
}

// SecretHasher は setup/bootstrap secret の保存用 hash を生成する port である。
//
// 役割:
//   - application service が bcrypt など platform 実装へ直接依存せず、secret 保存形式を runtime composition から受け取れるようにする。
//   - 平文 secret はこの port 呼び出し中だけ扱い、戻り値は保存用 hash のみに限定する。
//
// 引数:
//   - secretValue: setup token などの平文 secret。実装は前後空白の扱いと空値拒否を安全に処理する。
//
// 戻り値:
//   - string: 保存用 hash。
//   - error: secret が不正、または hash 生成に失敗した場合。
type SecretHasher interface {
	HashSecret(secretValue string) (string, error)
}

// SecretVerifier は保存済み secret hash と提示 secret を照合する port である。
//
// 役割:
//   - application service が bcrypt 比較実装へ直接依存せず、bootstrap/setup token の照合能力だけを受け取れるようにする。
//   - 照合失敗理由は bool に畳み、平文 secret や hash 構造を application error へ含めない。
//
// 引数:
//   - hash: 設定または DB から読み込んだ保存済み hash。
//   - secretValue: request や secure delivery flow から提示された平文 secret。
//
// 戻り値:
//   - bool: 保存済み hash と提示 secret が一致した場合だけ true。
type SecretVerifier interface {
	MatchesSecret(hash string, secretValue string) bool
}

// OperatorSessionIssuer は setup 完了直後に Operator session を発行する auth service 境界である。
//
// 役割:
//   - passkey login や setup transaction が認証済みにした OperatorID だけを受け取り、session 発行の詳細を再実装しない。
//   - OperatorSessionService の IssueOperatorSession method だけへ依存し、保存先や signer の具象型を持ち込まない。
//
// 引数:
//   - ctx: session store への保存に使う cancellation context。
//   - input: 認証済み Operator の canonical ID を含む入力 DTO。
//
// 戻り値:
//   - authapplication.OperatorSessionResult: accessToken と HttpOnly refresh Cookie command を分離した session DTO。
//   - error: Operator が存在しない、inactive、未登録状態、または session 保存に失敗した場合の stable application error。
type OperatorSessionIssuer interface {
	IssueOperatorSession(ctx context.Context, input authapplication.IssueOperatorSessionInput) (authapplication.OperatorSessionResult, error)
}

// ─── Config DTO ────────────────────────────────────────────────────────────

// BootstrapConfig は初回 Operator setup gate の application DTO である。
//
// 役割:
//   - platform/config 型を application 層へ import せず、必要な primitive だけを runtime composition から受け取る。
//   - SecretHash は opaque hash だけを保持し、bootstrap secret 平文は use case 入力として一時的に比較する。
type BootstrapConfig struct {
	Enabled    bool
	SecretHash string
	ExpiresAt  time.Time
}
