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
