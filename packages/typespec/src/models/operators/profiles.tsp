import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace Admin;

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
