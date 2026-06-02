import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("JWT 形式のアクセストークンを使用した Bearer 認証。Authorization ヘッダーに `Bearer <JWT>` を付与すること。トークンは短命の JWT であり、subject・sessionID・iat・exp を含む。")
model BearerAuth {
  @doc("Http authentication")
  type: AuthType.http;

  @doc("bearer auth scheme")
  scheme: "bearer";
}

@doc("Canonical ULID string used for auth-owned resource and correlation identifiers.")
scalar UlidId extends string;

@doc("認証・認可 failure を Product/Admin の両 artifact で同じ語彙に正規化する分類。")
enum AuthFailureClassification {
  @doc("認証 credential が存在しない、または検証できない状態。")
  unauthenticated: "unauthenticated",

  @doc("session は存在するが期限切れ、失効、または refresh が必要な状態。")
  sessionExpired: "session-expired",

  @doc("Product account が停止されている状態。Admin operator inactive など service 固有の理由は同じ failure response で表す。")
  accountSuspended: "account-suspended",

  @doc("Cookie と body など複数 refresh credential が同時提示されたため fail-close した状態。")
  credentialAmbiguous: "credential-ambiguous",

  @doc("refresh credential が不正、再利用、期限切れ、または path/context 所属不一致で拒否された状態。")
  invalidRefreshCredential: "invalid-refresh-credential",

  @doc("内部依存の一時障害など、credential 由来ではない失敗。")
  internalError: "internal-error",
}

@doc("認証 session credential の発行方式。browser は cookie mode を使い、external API / mobile / CLI / SDK は bearer mode を使う。")
enum CredentialMode {
  @doc("HttpOnly refresh Cookie を backend が設定し、response body には refreshToken 平文を含めない browser 向け方式。")
  cookie: "cookie",

  @doc("accessToken と refreshToken を response body で返し、Cookie command を返さない external client 向け方式。")
  bearer: "bearer",
}

@doc("browser 内の context index に対する更新種別。token や Cookie 値を含めず、UI と refresh bootstrap の hint だけを更新する。")
enum ContextIndexUpdateAction {
  @doc("指定 authContextId の entry を server 検証済み metadata で作成または更新する。")
  upsert: "upsert",

  @doc("指定 authContextId の entry を削除する。")
  remove: "remove",

  @doc("現在の service surface に属する context index entry をすべて削除する。")
  clearSurface: "clear-surface",
}

@doc("browser context index に保存してよい非 secret の表示 hint。accessToken、refreshToken、Cookie 値、setup token は含めない。")
model ContextIndexDisplayHint {
  @doc("Account email や operator email など、UI 表示に使う非 secret label。")
  label: string;

  @doc("補助表示に使う任意の非 secret text。role や locale など、認可判断に使わない値だけを入れる。")
  secondaryLabel?: string;
}

@doc("browser context index を更新するための非 secret hint。service artifact が Product/Admin context を決定するため、payload discriminator は持たない。")
model ContextIndexUpdateHint {
  @doc("context index に対して行う操作。")
  action: ContextIndexUpdateAction;

  @doc("操作対象の auth context。clear-surface では省略できる。")
  authContextId?: UlidId;

  @doc("対象 session の canonical ULID。upsert 時だけ返す。")
  sessionId?: UlidId;

  @doc("token を含まない UI 表示用 hint。upsert 時だけ返す。")
  displayHint?: ContextIndexDisplayHint;

  @doc("context が最後に server で確認された時刻。")
  lastSeenAt?: utcDateTime;

  @doc("context index entry を削除する目安時刻。認可可否は refresh response で再検証する。")
  expiresHintAt?: utcDateTime;
}

@doc("HttpOnly refresh Cookie を削除するために backend が返す command。transport adapter はこの内容から Set-Cookie Max-Age=0 を組み立てる。")
model CookieClearCommand {
  @doc("削除対象の Cookie 名。")
  name: string;

  @doc("削除対象 Cookie の exact Path。context-scoped refresh path `/api/v1/auth/contexts/{authContextId}/refresh` を返す。")
  path: string;

  @doc("Cookie 削除を表す Max-Age 値。")
  maxAge: 0;

  @doc("削除対象 Cookie が HttpOnly 属性を持つこと。")
  httpOnly: true;

  @doc("削除対象 Cookie が Secure 属性を持つこと。")
  secure: true;

  @doc("削除対象 Cookie の SameSite 属性。")
  sameSite: "Lax";

  @doc("削除対象 Cookie が束縛されていた auth context。")
  authContextId: UlidId;
}

@doc("認証・認可 failure の共通 response。requestId と normalized error だけを返し、token や Cookie 値は含めない。")
model AuthFailureResponse {
  @doc("request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("client が fail-close 処理を選ぶための normalized failure classification。")
  error: AuthFailureClassification;
}

@doc("入力不正や操作競合など、認証 credential 検証以外の operation error response。")
model AuthOperationErrorResponse {
  @doc("request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("client が表示・分岐に使う error code。secret や token 値は含めない。")
  error: string;
}
