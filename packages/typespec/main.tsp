import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

// Product と Admin の両 service から参照する route 非依存の共通 model を先に読み込む。
import "./src/common/errors.tsp";
import "./src/models/account_settings.tsp";
import "./src/models/admin.tsp";
import "./src/models/auth.tsp";
import "./src/models/status.tsp";

// Product service にだけ所属する /api/v1 route namespace を読み込む。
import "./src/routes/v1/_namespace.tsp";
import "./src/routes/v1/app_namespace.tsp";
import "./src/routes/v1/account_settings.tsp";
import "./src/routes/v1/auth.tsp";
import "./src/routes/v1/status.tsp";

// Admin service にだけ所属する /api/v1 route namespace を Product とは別に読み込む。
import "./src/routes/admin-v1/_namespace.tsp";
import "./src/routes/admin-v1/accounts.tsp";
import "./src/routes/admin-v1/auth.tsp";

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
