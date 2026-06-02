import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace Admin;

@doc("Admin operator の authorization concept を所有する marker model。RBAC の詳細は backend application が評価し、契約には role と operator profile だけを公開する。")
model AdminAuthorizationBoundary {
  @doc("authorization 判定対象 operator の canonical ULID。")
  operatorId: WWWTemplate.UlidId;
}
