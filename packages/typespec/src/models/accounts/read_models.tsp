import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("Product Account が表示と通知に使用する保存済みロケール。運用者向け設定や画面一時状態ではなく、AccountSetting.locale の値だけを表す。")
enum AccountLocale {
  @doc("日本語表示と日本語通知に使用する Product AccountSetting locale。")
  ja: "ja",

  @doc("英語表示と英語通知に使用する Product AccountSetting locale。")
  en: "en",
}

@doc("Product Account に属する現在の設定。AccountSetting は Account の表示・通知設定を表し、Auth 情報や運用者向け設定を含まない。")
model AccountSetting {
  @doc("現在の Product Account に保存されている表示・通知ロケール。対応値は AccountLocale の ja または en のみ。")
  locale: AccountLocale;
}

@doc("認証処理の合成結果として返す Product AccountSetting のスナップショット。refresh token rotation 自体は Auth が担当し、この snapshot は確定した AccountID から AccountSetting を読み込んだ結果だけを表す。")
model AccountSettingSnapshot {
  @doc("refresh response の合成時点で Product Account に保存されていた表示・通知ロケール。")
  locale: AccountLocale;
}

@doc("現在の Product Account の AccountSetting を返すレスポンス。requestId は追跡用であり、setting には Account 本人の保存済み設定だけを含める。")
model AccountSettingResponse {
  @doc("AccountSetting 取得または更新 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("現在の Product Account に属する保存済み AccountSetting。")
  setting: AccountSetting;
}

@doc("現在の Product Account の AccountSetting.locale を更新するリクエスト。Account ID は bearer session から確定するため body には含めない。")
model UpdateAccountSettingRequest {
  @doc("保存したい Product AccountSetting.locale。対応値以外は API 実装で拒否し、永続値を変更しない。")
  locale: AccountLocale;
}

@doc("Admin surface で表示・監査に使う Product Account の状態。Product domain の永続値を Admin API の read model として表すだけで、Admin 固有状態を混ぜない。")
enum AccountStatus {
  @doc("Product Account が利用可能な状態。")
  active: "active",

  @doc("Product Account が停止され、既存 session の失効境界を持つ状態。")
  suspended: "suspended",
}

@doc("Account 一覧と詳細で返す Product Account の要約。Admin route から参照できる account concept model であり、Admin surface 固有の catch-all DTO ではない。")
model AccountSummary {
  @doc("対象 Product Account の canonical ULID。")
  accountId: UlidId;

  @doc("Product Account の canonical email address。")
  email: string;

  @doc("API が観測した Product Account lifecycle state。")
  status: AccountStatus;

  @doc("Product Account が作成された日時。")
  createdAt: utcDateTime;

  @doc("対象 Product Account に登録済みの passkey 数。")
  passkeyCount: int32;
}

@doc("operator が customer account を作成するときの request body。email と初期 locale だけを受け取り、Account 不変条件は Go backend domain object が検証する。")
model CreateAccountRequest {
  @doc("作成対象 Product Account の email address。")
  @format("email")
  email: string;

  @doc("作成時に保存する Product AccountSetting locale。未指定時の default は backend application が決定する。")
  locale?: AccountLocale;
}

@doc("account 一覧 API の response。requestId は監査・追跡に使用し、accounts は account concept read model として返す。")
model AccountListResponse {
  @doc("一覧取得 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("検索条件に一致した Product Account の要約一覧。")
  accounts: AccountSummary[];

  @doc("次ページが存在するときだけ返す opaque cursor。")
  nextCursor?: string;
}

@doc("account 詳細 API の response。")
model AccountDetailResponse {
  @doc("詳細取得 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("対象 Product Account の read model。")
  account: AccountSummary;
}

@doc("account 作成 API の response。作成された Product Account と、対応する audit event の相関 ID だけを返す。")
model CreateAccountResponse {
  @doc("作成 request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("作成された Product Account の read model。")
  account: AccountSummary;

  @doc("Account 作成 intent/outcome を記録する Admin audit event の canonical ULID。")
  auditEventId: UlidId;
}
