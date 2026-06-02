import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace Admin;

@doc("Admin operator passkey login ceremony を開始する request。")
model AdminPasskeyStartRequest {
  @doc("Admin operator を識別する email address。")
  identifier: string;
}

@doc("Admin operator passkey login ceremony を完了する request。")
model AdminPasskeyFinishRequest {
  @doc("start response と対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("session credential の発行方式。Admin Console は cookie、automation client は bearer を指定する。")
  credentialMode: WWWTemplate.CredentialMode;

  @doc("browser WebAuthn API から返された assertion credential。")
  credential: WWWTemplate.WebAuthnAssertionCredential;
}

@doc("初回 Admin operator 作成 ceremony を開始する request。bootstrapSecret は backend で opaque hash と期限を検証し、response body や永続化先へ出さない。")
model AdminInitialSetupStartRequest {
  @doc("初回 admin operator として登録する canonical email address。")
  @format("email")
  email: string;

  @doc("WebAuthn user displayName に使う表示名。未指定や空白の扱いは backend application が安全に正規化する。")
  displayName: string;

  @doc("初回 setup gate を通過するための bootstrap secret。")
  bootstrapSecret: string;
}

@doc("初回 Admin operator 作成 ceremony を完了する request。bootstrapSecret は finish 時にも再検証し、start 後の競合や期限切れを fail-close にする。")
model AdminInitialSetupFinishRequest {
  @doc("初回 admin operator として登録する canonical email address。")
  @format("email")
  email: string;

  @doc("WebAuthn user displayName に使う表示名。")
  displayName: string;

  @doc("初回 setup gate を通過するための bootstrap secret。")
  bootstrapSecret: string;

  @doc("start response と対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("session credential の発行方式。Admin Console は cookie、automation client は bearer を指定する。")
  credentialMode: WWWTemplate.CredentialMode;

  @doc("browser WebAuthn API から返された attestation credential。")
  credential: WWWTemplate.WebAuthnAttestationCredential;
}

@doc("Admin operator 作成 API の request。setup token は secure delivery port だけで配送し、response body には含めない。")
model AdminCreateOperatorRequest {
  @doc("作成対象 Admin operator の email address。")
  @format("email")
  email: string;

  @doc("作成対象 Admin operator の role。")
  role: AdminOperatorRole;
}

@doc("setup token delivery の結果。平文 token ではなく配送状態だけを表す。")
enum AdminSetupTokenDeliveryStatus {
  @doc("setup token が secure delivery port で配送された。")
  sent: "sent",
}

@doc("Admin operator 作成 API の response。operator summary、delivery status、audit correlation だけを返し、setup token 平文は含めない。")
model AdminCreateOperatorResponse {
  @doc("作成 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("作成された Admin operator の profile。")
  operator: AdminOperatorProfile;

  @doc("setup token の配送結果。")
  deliveryStatus: AdminSetupTokenDeliveryStatus;

  @doc("Operator 作成 intent/outcome を記録する Admin audit event の canonical ULID。")
  auditEventId: WWWTemplate.UlidId;
}

@doc("Admin operator 初期設定 ceremony を開始する request。setupToken は backend が hash と期限で検証し、response body へ再露出しない。")
model AdminOperatorSetupStartRequest {
  @doc("operator setup に使う一回限りの token。")
  setupToken: string;
}

@doc("Admin operator 初期設定 ceremony を完了する request。")
model AdminOperatorSetupFinishRequest {
  @doc("operator setup に使う一回限りの token。")
  setupToken: string;

  @doc("start response と対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("session credential の発行方式。Admin Console は cookie、automation client は bearer を指定する。")
  credentialMode: WWWTemplate.CredentialMode;

  @doc("browser WebAuthn API から返された attestation credential。")
  credential: WWWTemplate.WebAuthnAttestationCredential;
}

@doc("Admin Console の operator session response。operator refreshToken は HttpOnly Cookie だけで扱い、body には含めない。")
model AdminOperatorSessionResponse {
  ...WWWTemplate.CookieAuthEnvelope;

  @doc("認証された Admin operator の profile。")
  operator: AdminOperatorProfile;
}

@doc("Admin automation client の Bearer mode operator session response。Cookie command や browser context index hint を含めない。")
model AdminBearerOperatorSessionResponse {
  ...WWWTemplate.BearerAuthEnvelope;

  @doc("認証された Admin operator の profile。")
  operator: AdminOperatorProfile;
}

@doc("Admin login / setup が返す credential mode 別 session response。Cookie mode と Bearer mode の body shape を shared auth envelope から構成する。")
@oneOf
union AdminOperatorAuthSessionResponse {
  cookie: AdminOperatorSessionResponse,
  bearer: AdminBearerOperatorSessionResponse,
}

@doc("Admin Console context refresh 成功時の response。refreshToken 平文を body に含めず、path-scoped Cookie rotation を transport header に委ねる。")
model AdminContextRefreshResponse {
  ...WWWTemplate.CookieAuthEnvelope;

  @doc("認証された Admin operator の profile。")
  operator: AdminOperatorProfile;
}

@doc("Admin automation client の Bearer mode context refresh 成功 response。Cookie command や browser context index hint を含めない。")
model AdminBearerContextRefreshResponse {
  ...WWWTemplate.BearerAuthEnvelope;

  @doc("認証された Admin operator の profile。")
  operator: AdminOperatorProfile;
}

@doc("Admin context refresh の credential mode 別 success response。Cookie mode と Bearer mode は同じ auth envelope field 名を使う。")
@oneOf
union AdminOperatorContextRefreshResponse {
  cookie: AdminContextRefreshResponse,
  bearer: AdminBearerContextRefreshResponse,
}
