import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace Admin;

@doc("Admin surface で表示・監査に使う Product Account の状態。Product domain の永続値を Admin API の read model として表すだけで、Admin 固有状態を混ぜない。")
enum AdminAccountStatus {
  @doc("Product Account が利用可能な状態。")
  active: "active",

  @doc("Product Account が停止され、既存 session の失効境界を持つ状態。")
  suspended: "suspended",
}

@doc("Admin account 一覧と詳細で返す Product Account の要約。Admin route から参照する read model であり、Product route namespace には所属しない。")
model AdminAccountSummary {
  @doc("対象 Product Account の canonical ULID。")
  accountId: WWWTemplate.UlidId;

  @doc("Product Account の canonical email address。")
  email: string;

  @doc("Admin API が観測した Product Account lifecycle state。")
  status: AdminAccountStatus;

  @doc("Product Account が作成された日時。")
  createdAt: utcDateTime;

  @doc("対象 Product Account に登録済みの passkey 数。")
  passkeyCount: int32;
}

@doc("Admin operator が customer account を作成するときの request body。email と初期 locale だけを受け取り、Account 不変条件は Go backend domain object が検証する。")
model AdminCreateAccountRequest {
  @doc("作成対象 Product Account の email address。")
  @format("email")
  email: string;

  @doc("作成時に保存する Product AccountSetting locale。未指定時の default は backend application が決定する。")
  locale?: WWWTemplate.AccountLocale;
}

@doc("Admin account 一覧 API の response。requestId は監査・追跡に使用し、accounts は Product route から独立した Admin read model として返す。")
model AdminAccountListResponse {
  @doc("一覧取得 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("検索条件に一致した Product Account の要約一覧。")
  accounts: AdminAccountSummary[];

  @doc("次ページが存在するときだけ返す opaque cursor。")
  nextCursor?: string;
}

@doc("Admin account 詳細 API の response。")
model AdminAccountDetailResponse {
  @doc("詳細取得 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("対象 Product Account の Admin read model。")
  account: AdminAccountSummary;
}

@doc("Admin account 作成 API の response。作成された Product Account と、対応する audit event の相関 ID だけを返す。")
model AdminCreateAccountResponse {
  @doc("作成 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("作成された Product Account の Admin read model。")
  account: AdminAccountSummary;

  @doc("Account 作成 intent/outcome を記録する Admin audit event の canonical ULID。")
  auditEventId: WWWTemplate.UlidId;
}

@doc("Admin operator の role。UI 表示用の分類であり、最終的な authorization は Admin backend が行う。")
enum AdminOperatorRole {
  @doc("すべての Admin 権限を持つ operator。")
  admin: "admin",

  @doc("通常の運用操作を担当する operator。")
  operator: "operator",

  @doc("閲覧だけを許可される operator。")
  viewer: "viewer",
}

@doc("Admin operator の現在状態を返す read model。")
model AdminOperatorProfile {
  @doc("Admin operator の canonical ULID。")
  operatorId: WWWTemplate.UlidId;

  @doc("Admin operator の canonical email address。")
  email: string;

  @doc("Admin operator に割り当てられた role。")
  role: AdminOperatorRole;

  @doc("Admin operator が有効な場合 true。inactive operator は backend で認証・認可されない。")
  active: boolean;
}

@doc("Admin operator passkey login ceremony を開始する request。")
model AdminPasskeyStartRequest {
  @doc("Admin operator を識別する email address。")
  identifier: string;
}

@doc("Admin operator passkey login ceremony を完了する request。")
model AdminPasskeyFinishRequest {
  @doc("start response と対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

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

  @doc("browser WebAuthn API から返された attestation credential。")
  credential: WWWTemplate.WebAuthnAttestationCredential;
}

@doc("Admin operator auth 成功時の response。refreshToken は HttpOnly Cookie で扱うため body には含めない。")
model AdminAuthSessionResponse {
  @doc("認証 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("認証された Admin operator の profile。")
  operator: AdminOperatorProfile;

  @doc("Admin operator session の canonical ULID。")
  sessionId: WWWTemplate.UlidId;

  @doc("短命の Admin operator access token。")
  accessToken: string;

  @doc("accessToken の失効日時。")
  expiresAt: utcDateTime;

  @doc("mutation request で session と照合する CSRF token。")
  csrfToken: string;
}

@doc("現在の Admin operator を返す response。")
model AdminCurrentOperatorResponse {
  @doc("current operator request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("現在の Admin operator profile。")
  operator: AdminOperatorProfile;
}

@doc("Admin operator 自身が登録している passkey credential の表示用要約。credential handle や public key は秘匿し、管理操作に必要な識別子と時刻だけを返す。")
model AdminOperatorPasskeyItem {
  @doc("Admin operator passkey credential の canonical ULID。削除 API の path parameter として使う。")
  id: WWWTemplate.UlidId;

  @doc("passkey credential が Admin schema に登録された日時。")
  createdAt: utcDateTime;

  @doc("最後にこの passkey credential が認証に使われた日時。未使用の場合は返さない。")
  lastUsedAt?: utcDateTime;
}

@doc("Admin operator 自身の passkey credential 一覧 response。Product account passkey model を使わず、Admin operator credential だけを表す。")
model AdminOperatorPasskeyListResponse {
  @doc("一覧取得 request に対応する canonical ULID の追跡 ID。")
  requestId: WWWTemplate.UlidId;

  @doc("現在の Admin operator に登録されている passkey credential 一覧。")
  passkeys: AdminOperatorPasskeyItem[];
}
