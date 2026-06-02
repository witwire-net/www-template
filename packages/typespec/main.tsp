import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

// Product と Admin の両 service から参照する route 非依存の共通 concept model を先に読み込む。
import "./src/common/errors.tsp";
import "./src/models/accounts/read_models.tsp";
import "./src/models/auth/logout.tsp";
import "./src/models/auth/primitives.tsp";
import "./src/models/auth/recovery.tsp";
import "./src/models/auth/refresh.tsp";
import "./src/models/auth/sessions.tsp";
import "./src/models/auth/webauthn.tsp";
import "./src/models/operators/authorization.tsp";
import "./src/models/operators/profiles.tsp";
import "./src/models/operators/setup.tsp";
import "./src/models/status.tsp";

// Product service にだけ所属する /api/v1 route namespace を読み込む。context-scoped refresh は Admin と同じ相対 path を持つが、ここで import tree を分けることで Product artifact へ Admin route を混入させない。
import "./src/routes/v1/product/_namespace.tsp";
import "./src/routes/v1/product/account_settings.tsp";
import "./src/routes/v1/product/app_namespace.tsp";
import "./src/routes/v1/product/auth.tsp";
import "./src/routes/v1/product/status.tsp";

// Admin service にだけ所属する /api/v1 route namespace を Product とは別に読み込む。Product と同名の context refresh path も Admin service 側だけに生成し、origin / SDK / Go bindings の分離を保つ。
import "./src/routes/v1/admin/_namespace.tsp";
import "./src/routes/v1/admin/accounts.tsp";
import "./src/routes/v1/admin/auth.tsp";

using Http;
using TypeSpec.OpenAPI;

@service(#{ title: "www-template Admin API" })
@info(#{ version: "1.0.0" })
@server("https://admin.www-template.example", "Admin API server")
namespace Admin {

}

@service(#{ title: "www-template API" })
@info(#{ version: "1.0.0" })
@server("https://product.www-template.example", "Product API server")
namespace WWWTemplate {

}
