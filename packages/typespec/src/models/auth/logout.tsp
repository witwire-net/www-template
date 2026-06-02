import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("logout / revoke 処理の共通 response。accessToken claims で確定した context の Cookie clear と context index cleanup だけを返す。")
model LogoutResponse {
  @doc("logout / revoke request に対応する canonical ULID の追跡 ID。")
  requestId: UlidId;

  @doc("対象 session または refresh family が失効されたことを示す。")
  revoked: true;

  @doc("revocation 対象 refresh Cookie を exact Path で削除する command 一覧。")
  clearCookieCommands: CookieClearCommand[];

  @doc("revocation 後に browser context index から対象 entry を削除するための hint。")
  contextIndexUpdateHints: ContextIndexUpdateHint[];
}
