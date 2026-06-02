import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("Cookie mode session / refresh response が共有する auth envelope。Product/Admin はこの単一定義を spread して同じ field 名を公開する。")
model CookieAuthEnvelope {
  @doc("認証 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("browser cookie mode で発行されたことを示す discriminator。")
  credentialMode: "cookie";

  @doc("refresh path と session metadata を束縛する auth context。")
  authContextId: UlidId;

  @doc("対象 session の canonical ULID。")
  sessionId: UlidId;

  @doc("protected API に Authorization Bearer として送信する短命 accessToken。")
  accessToken: string;

  @doc("accessToken の失効日時。")
  expiresAt: utcDateTime;

  @doc("browser の origin-local context index を更新する非 secret hint。")
  contextIndexUpdateHints: ContextIndexUpdateHint[];

  @doc("logout / revoke / suspend などで削除すべき旧 refresh Cookie がある場合の command。通常 login では空配列。")
  clearCookieCommands: CookieClearCommand[];
}

@doc("Bearer mode session / refresh response が共有する auth envelope。Cookie command や browser context index hint を持たない。")
model BearerAuthEnvelope {
  @doc("認証 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("external bearer mode で発行されたことを示す discriminator。")
  credentialMode: "bearer";

  @doc("refreshToken と session metadata を束縛する auth context。")
  authContextId: UlidId;

  @doc("対象 session の canonical ULID。")
  sessionId: UlidId;

  @doc("protected API に Authorization Bearer として送信する短命 accessToken。")
  accessToken: string;

  @doc("external client が body で保持し、context refresh で rotation する opaque refreshToken。")
  refreshToken: string;

  @doc("accessToken の失効日時。")
  expiresAt: utcDateTime;
}

@doc("Product auth response の account subject payload。service artifact が Product 文脈を決めるため principal discriminator は持たない。")
model ProductAccountSubject {
  @doc("対象 Product account の canonical ULID。")
  accountId: UlidId;

  @doc("認証に使われた passkey credential の canonical ULID。")
  passkeyCredentialId?: UlidId;
}

@doc("Product browser cookie mode の session response。refreshToken は HttpOnly Cookie だけで扱い、body には含めない。")
model ProductCookieAuthSessionResponse {
  ...CookieAuthEnvelope;

  @doc("認証された Product account の subject payload。")
  account: ProductAccountSubject;

  @doc("login / refresh 後に Product UI が account 設定を同期するための snapshot。")
  accountSetting?: AccountSettingSnapshot;
}

@doc("Product external bearer mode の session response。Cookie command や browser context index hint を含めず、body で refreshToken を返す。")
model ProductBearerAuthSessionResponse {
  ...BearerAuthEnvelope;

  @doc("認証された Product account の subject payload。")
  account: ProductAccountSubject;

  @doc("login / refresh 後に external client が必要に応じて参照できる Product 設定 snapshot。")
  accountSetting?: AccountSettingSnapshot;
}

@doc("Product login / register が返す credential mode 別 session response。Cookie mode と Bearer mode の body shape を分離する。")
@oneOf
union ProductAuthSessionResponse {
  cookie: ProductCookieAuthSessionResponse,
  bearer: ProductBearerAuthSessionResponse,
}

@doc("ログイン中のセッション（デバイス）を表すアイテム。各セッションは一意の sessionId を持つ。")
model SessionItem {
  sessionId: UlidId;
  deviceName: string;
  loginAt: utcDateTime;
  lastActiveAt: utcDateTime;
  ipHash: string;
  isCurrentSession: boolean;
}

@doc("ログイン中のセッション一覧レスポンス。認証済みアカウントが所有する全セッションを含む。")
model SessionListResponse {
  requestId: UlidId;
  sessions: SessionItem[];
}
