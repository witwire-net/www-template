package config

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	secrethash "www-template/packages/backend/internal/platform/secret"
)

// AdminCookieConfig は Admin refreshToken Cookie の runtime 設定を保持する。
//
// 役割:
//   - Admin operator auth が発行する HttpOnly Cookie の名前、domain、path、SameSite、Secure 属性を surface 専用設定として表現する。
//   - Product Cookie 設定と共有せず、Admin startup validation が Admin domain と cookie domain の不一致を fail-close で検出できるようにする。
//
// 各フィールド:
//   - Name: ブラウザーへ設定する Admin refreshToken Cookie 名。
//   - Domain: Cookie を送る Admin host。Admin runtime の Domain と一致しなければならない。
//   - Path: Cookie path。Admin API 全体へ限定するため `/` を要求する。
//   - Secure: production で HTTPS のみへ送信するための Secure 属性。
//   - SameSite: CSRF 境界を固定する SameSite 属性。現設計では `Lax` だけを許可する。
//
// 利用例:
//
//	cookie := config.AdminCookieConfig{Name: "www_template_admin_refresh", Domain: "admin.example.com", Path: "/", Secure: true, SameSite: "Lax"}
type AdminCookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	SameSite string
}

// AdminDatabaseConfig は Admin backend が DB 内 Admin-owned schema を扱うための runtime 設定を保持する。
//
// 役割:
//   - Admin runtime が使用する least-privilege DB role 名を runtime 設定から分離して保持する。
//   - startup validation で role の未設定や危険な文字を拒否し、誤った DB role で Admin surface を起動しないようにする。
//
// 各フィールド:
//   - Role: Admin schema と account management に必要な権限だけを付与された DB role 名。
//
// 利用例:
//
//	database := config.AdminDatabaseConfig{Role: "admin_console_write"}
type AdminDatabaseConfig struct {
	Role string
}

// AdminBootstrapConfig は初回 Admin operator 作成を一時的に許可する bootstrap gate 設定である。
//
// 役割:
//   - Operator が 0 件の環境だけで初回 admin 作成を許可するため、明示的な enable flag、secret hash、有効期限を保持する。
//   - bootstrap secret 平文は設定にも runtime state にも保持せず、opaque hash だけを起動時 validation と setup use case へ渡す。
//   - production では enable=true のまま期限や hash が欠けた状態を拒否し、初回作成 route の fail-open を防ぐ。
//
// 各フィールド:
//   - Enabled: 初回 setup route を許可する一時的な gate。
//   - SecretHash: bootstrap secret の opaque hash。平文 secret は保持しない。
//   - ExpiresAt: bootstrap gate の失効日時。現在時刻を過ぎると setup は拒否される。
//
// 利用例:
//
//	bootstrap := config.AdminBootstrapConfig{Enabled: true, SecretHash: "$2a$10$examplebcryptbootstraphash", ExpiresAt: time.Now().Add(time.Hour)}
type AdminBootstrapConfig struct {
	Enabled    bool
	SecretHash string
	ExpiresAt  time.Time
}

// AdminRuntimeConfig は Admin API binary 専用の surface 設定を保持する。
//
// 役割:
//   - Admin domain、Product domain、Admin Cookie、Admin runtime DB role、Admin Valkey URL、Product Valkey URL を明示的に分離する。
//   - Admin startup が Product と Admin の surface 境界を fail-close で検証するための入力を一箇所へ集約する。
//
// 各フィールド:
//   - Domain: Admin frontend と Admin backend が共有する公開 origin。
//   - ProductDomain: Product surface の公開 origin。Admin domain と一致してはならない。
//   - Cookie: Admin operator refreshToken Cookie の属性。
//   - Database: Admin-owned DB role の属性。
//   - Valkey: Admin operator auth state 用 Valkey 接続 URL と key prefix。
//   - ProductValkey: Product runtime が使う Valkey 接続 URL。Admin runtime から接続せず、logical DB 衝突検証だけに使う。
//
// 利用例:
//
//	admin := config.AdminRuntimeConfig{Domain: "https://admin.example.com", ProductDomain: "https://app.example.com", Database: config.AdminDatabaseConfig{Role: "admin_console_write"}}
type AdminRuntimeConfig struct {
	Domain        string
	ProductDomain string
	Cookie        AdminCookieConfig
	Database      AdminDatabaseConfig
	Bootstrap     AdminBootstrapConfig
	Valkey        ValkeyConfig
	ProductValkey ValkeyConfig
}

// ValidateAdminRuntime は Admin API binary 専用の runtime 設定を fail-close で検証する。
//
// 役割:
//   - Product Config.Validate では扱わない Admin surface 固有値を起動前に検査する。
//   - Admin domain / Product domain の混同、Cookie domain の不一致、Admin runtime DB role の未設定、Admin Valkey URL の未設定を拒否する。
//   - Admin と Product の Valkey が同じ endpoint / logical DB を指す場合、operator auth state の混在を防ぐため拒否する。
//   - production では Admin / Product domain の HTTPS と安全な host、および Admin Cookie の Secure 属性を必須にする。
//
// 戻り値:
//   - nil: Admin runtime 設定が起動可能な境界を満たす。
//   - error: どの設定値が拒否されたかを列挙した startup error。
//
// 利用例:
//
//	if err := cfg.ValidateAdminRuntime(); err != nil { return err }
func (c Config) ValidateAdminRuntime() error {
	// Step 1: Admin / Product origin を先に正規化し、後続の cookie domain 検証で同じ解釈を使う。
	_, adminHost, adminErr := c.validateAdminOrigin(c.Admin.Domain, "admin.domain")
	_, productHost, productErr := c.validateAdminOrigin(c.Admin.ProductDomain, "admin.product_domain")

	// Step 2: 各 validator が検出した問題を集約し、startup caller が一度で全欠落を把握できるようにする。
	errs := make([]string, 0)
	errs = appendAdminValidationError(errs, adminErr)
	errs = appendAdminValidationError(errs, productErr)
	errs = validateAdminDomainSeparation(errs, adminHost, productHost)
	errs = c.validateAdminCookie(errs, adminHost)
	errs = c.validateAdminDatabase(errs)
	errs = c.validateAdminBootstrap(errs)
	errs = c.validateAdminValkey(errs)
	errs = c.validateAdminOpenSearch(errs)

	// Step 3: 一つでも問題があれば Admin surface を起動しない fail-close error として返す。
	if len(errs) > 0 {
		return errors.New("admin runtime config validation failed: " + strings.Join(errs, "; "))
	}
	return nil
}

// validateAdminBootstrap は初回 Admin setup gate の安全な設定を検証する。
func (c Config) validateAdminBootstrap(errs []string) []string {
	// Step 1: bootstrap が無効な場合は route 側が常に拒否するため、secret hash と期限の検査は不要にする。
	bootstrap := c.Admin.Bootstrap
	if !bootstrap.Enabled {
		return errs
	}

	// Step 2: enable=true で hash が空または bcrypt 形式でない場合は、起動時に fail-close する。
	trimmedSecretHash := strings.TrimSpace(bootstrap.SecretHash)
	if trimmedSecretHash == "" {
		errs = append(errs, "admin.bootstrap.secret_hash is required when bootstrap is enabled")
	} else if !secrethash.IsBcryptHash(trimmedSecretHash) {
		errs = append(errs, "admin.bootstrap.secret_hash must be a bcrypt hash")
	}

	// Step 3: 期限が未設定または起動時点で失効済みなら初回 setup を許可しない。
	if bootstrap.ExpiresAt.IsZero() {
		errs = append(errs, "admin.bootstrap.expires_at is required when bootstrap is enabled")
	} else if !time.Now().UTC().Before(bootstrap.ExpiresAt.UTC()) {
		errs = append(errs, "admin.bootstrap.expires_at must be in the future when bootstrap is enabled")
	}
	return errs
}

// validateAdminOpenSearch は Admin audit projection 用 OpenSearch namespace が Product namespace と衝突しないことを検証する。
func (c Config) validateAdminOpenSearch(errs []string) []string {
	// Step 1: Admin audit projection は index prefix で namespace を固定するため、空値は誤った wildcard 書き込みを避ける目的で拒否する。
	adminPrefix := strings.TrimSpace(c.Infra.OpenSearch.AdminAuditIndexPrefix)
	if adminPrefix == "" {
		errs = append(errs, "opensearch.admin_audit_index_prefix is required")
	}

	// Step 2: Product domain prefix も衝突検査の比較対象として必須にし、Admin runtime が Product namespace を知らない状態で起動しないようにする。
	productPrefix := strings.TrimSpace(c.Infra.OpenSearch.ProductIndexPrefix)
	if productPrefix == "" {
		errs = append(errs, "opensearch.product_index_prefix is required")
	}
	if adminPrefix == "" || productPrefix == "" {
		return errs
	}

	// Step 3: 同一 prefix または包含関係は wildcard index pattern で交差するため、startup 時点で fail-close する。
	adminComparable := strings.ToLower(adminPrefix)
	productComparable := strings.ToLower(productPrefix)
	if adminComparable == productComparable || strings.Contains(adminComparable, productComparable) || strings.Contains(productComparable, adminComparable) {
		return append(errs, "opensearch.admin_audit_index_prefix must not collide with opensearch.product_index_prefix")
	}
	return errs
}

// validateAdminOrigin は Admin runtime で扱う origin 文字列を検証し、正規化 origin と host を返す。
func (c Config) validateAdminOrigin(rawOrigin string, field string) (string, string, error) {
	// Step 1: 空文字列は domain 境界が定義されていないため、即座に fail-close 対象にする。
	trimmed := strings.TrimSpace(rawOrigin)
	if trimmed == "" {
		return "", "", errors.New(field + " is required")
	}

	// Step 2: origin として解釈できる URL かを確認し、path / query / fragment を持つ URL を拒否する。
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", errors.New(field + " is invalid URL: " + trimmed)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", errors.New(field + " must use http or https origin: " + trimmed)
	}
	if parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.New(field + " must be an origin without path, query, or fragment: " + trimmed)
	}

	// Step 3: production では既存の Product origin policy と同じ HTTPS / host 安全性を Admin surface に適用する。
	if c.Environment != "development" {
		if err := validateProductionOrigin(trimmed); err != nil {
			return "", "", errors.New(field + ": " + err.Error())
		}
	}

	// Step 4: 比較用に scheme と host を小文字化し、末尾 slash の有無で別 origin と誤判定しないようにする。
	normalized := strings.ToLower(parsed.Scheme + "://" + parsed.Host)
	return normalized, parsed.Hostname(), nil
}

// validateAdminDomainSeparation は Admin と Product の domain host が一致しないことを検証する。
func validateAdminDomainSeparation(errs []string, adminHost string, productHost string) []string {
	// Step 1: どちらかの origin validation が失敗している場合、重複 error を避けるため一致判定を行わない。
	if adminHost == "" || productHost == "" {
		return errs
	}

	// Step 2: Admin と Product が同じ domain host だと port が違っても surface 分離が崩れるため、startup を拒否する。
	if strings.EqualFold(adminHost, productHost) {
		return append(errs, "admin.domain must differ from admin.product_domain")
	}
	return errs
}

// validateAdminCookie は Admin Cookie 属性が Admin domain と一致し、安全な固定値を持つことを検証する。
func (c Config) validateAdminCookie(errs []string, adminHost string) []string {
	// Step 1: Cookie 名がないと refreshToken の保管先が曖昧になるため拒否する。
	cookie := c.Admin.Cookie
	if strings.TrimSpace(cookie.Name) == "" {
		errs = append(errs, "admin.cookie.name is required")
	}

	// Step 2: Cookie domain を必須にし、Admin origin host と完全一致させて Product host への送信を防ぐ。
	cookieDomain := strings.TrimSpace(cookie.Domain)
	if cookieDomain == "" {
		errs = append(errs, "admin.cookie.domain is required")
	} else if adminHost != "" && !strings.EqualFold(cookieDomain, adminHost) {
		errs = append(errs, "admin.cookie.domain must match admin.domain host")
	}

	// Step 3: Admin API 全体で同じ refresh cookie を扱う前提を固定するため path は `/` のみ許可する。
	if strings.TrimSpace(cookie.Path) != "/" {
		errs = append(errs, "admin.cookie.path must be /")
	}

	// Step 4: [OpenSpec Task 4.47] SameSite=Lax cookie behavior を startup validation で固定し、CSRF policy の曖昧化を防ぐ。
	if !strings.EqualFold(strings.TrimSpace(cookie.SameSite), "Lax") {
		errs = append(errs, "admin.cookie.same_site must be Lax")
	}

	// Step 5: [OpenSpec Task 4.48] insecure production cookie rejection として、production では Secure=false の Cookie を拒否し、平文 transport へ refreshToken を出さない。
	if c.Environment != "development" && !cookie.Secure {
		errs = append(errs, "admin.cookie.secure is required outside development")
	}
	return errs
}

// validateAdminDatabase は Admin runtime 用 DB role 名が明示されていることを検証する。
func (c Config) validateAdminDatabase(errs []string) []string {
	// Step 1: Admin schema 用 least-privilege role が空だと runtime role の取り違えを検出できないため拒否する。
	role := strings.TrimSpace(c.Admin.Database.Role)
	if role == "" {
		return append(errs, "admin.database.role is required")
	}

	// Step 2: role 名として扱う値に空白や SQL 区切り文字が含まれる場合は誤設定または注入リスクとして拒否する。
	if strings.ContainsAny(role, " \t\n\r/\\'\";") {
		return append(errs, "admin.database.role contains invalid characters")
	}
	return errs
}

// validateAdminValkey は Admin operator auth state 用の Valkey URL が明示され、Product Valkey と logical DB が分離されていることを検証する。
func (c Config) validateAdminValkey(errs []string) []string {
	// Step 1: Admin auth state を保存する Valkey URL がない場合、operator session を安全に保持できないため拒否する。
	rawURL := strings.TrimSpace(c.Admin.Valkey.URL)
	var parsed *url.URL
	if rawURL == "" {
		errs = append(errs, "admin.valkey.url is required")
	} else {
		// Step 2: Redis/Valkey 互換 URL として最低限の scheme / host を検証し、後続 adapter が曖昧な値を受け取らないようにする。
		var err error
		parsed, err = url.Parse(rawURL)
		switch {
		case err != nil:
			errs = append(errs, "admin.valkey.url is invalid URL")
		case parsed.Scheme != "redis" && parsed.Scheme != "rediss":
			errs = append(errs, "admin.valkey.url must use redis or rediss scheme")
		case parsed.Host == "":
			errs = append(errs, "admin.valkey.url host is required")
		}
	}

	// Step 3: Product Valkey URL も必須にし、Admin operator auth state と Product auth state の logical DB 分離を証明できない設定を拒否する。
	productRawURL := strings.TrimSpace(c.Admin.ProductValkey.URL)
	if productRawURL == "" {
		return append(errs, "admin.product_valkey.url is required")
	}
	if parsed == nil {
		return errs
	}
	return validateAdminProductValkeySeparation(errs, parsed, productRawURL)
}

// validateAdminProductValkeySeparation は Admin / Product Valkey URL の endpoint と logical DB が一致しないことを検証する。
func validateAdminProductValkeySeparation(errs []string, adminURL *url.URL, productRawURL string) []string {
	// Step 1: Product URL も Redis/Valkey 互換 URL として解釈し、誤った値で fail-open しないよう startup validation error にする。
	productURL, err := url.Parse(productRawURL)
	if err != nil {
		return append(errs, "admin.product_valkey.url is invalid URL")
	}
	if productURL.Scheme != "redis" && productURL.Scheme != "rediss" {
		return append(errs, "admin.product_valkey.url must use redis or rediss scheme")
	}
	if productURL.Host == "" {
		return append(errs, "admin.product_valkey.url host is required")
	}

	// Step 2: endpoint と logical DB の正規化結果が一致する場合、Admin と Product の state が同じ DB に書かれるため拒否する。
	adminEndpoint, adminDB, adminOK := valkeyEndpointLogicalDB(adminURL)
	productEndpoint, productDB, productOK := valkeyEndpointLogicalDB(productURL)
	if !adminOK || !productOK {
		return append(errs, "admin.valkey.url and admin.product_valkey.url must use numeric logical DB paths")
	}
	if adminEndpoint == productEndpoint && adminDB == productDB {
		return append(errs, "admin.valkey.url must not share logical DB with admin.product_valkey.url")
	}
	return errs
}

// valkeyEndpointLogicalDB は Valkey URL を endpoint と logical DB 番号へ正規化する。
func valkeyEndpointLogicalDB(parsed *url.URL) (string, int, bool) {
	// Step 1: Redis/Valkey の logical DB は path で指定され、未指定または `/` の場合は DB 0 として扱う。
	dbPath := strings.TrimPrefix(parsed.EscapedPath(), "/")
	if dbPath == "" {
		dbPath = "0"
	}

	// Step 2: 数値以外や複数 segment の path は logical DB として曖昧なため、caller 側で validation error にできるよう false を返す。
	logicalDB, err := strconv.Atoi(dbPath)
	if err != nil || logicalDB < 0 || strings.Contains(dbPath, "/") {
		return "", 0, false
	}

	// Step 3: scheme や credential を除き、同じ Valkey endpoint かどうかを hostname と port の組で比較する。
	port := parsed.Port()
	if port == "" {
		port = "6379"
	}
	endpoint := strings.ToLower(parsed.Hostname() + ":" + port)
	return endpoint, logicalDB, true
}

// appendAdminValidationError は optional error を validation error list へ追加する。
func appendAdminValidationError(errs []string, err error) []string {
	// Step 1: nil error は成功した validator を示すため、既存 slice をそのまま返す。
	if err == nil {
		return errs
	}

	// Step 2: error message だけを集約し、上位の startup error で secret 値を出さずに理由を提示する。
	return append(errs, err.Error())
}
