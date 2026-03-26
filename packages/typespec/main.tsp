import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

// Shared models
import "./src/common/errors.tsp";
import "./src/models/auth.tsp";
import "./src/models/status.tsp";

// Versioned API routes
import "./src/routes/v1/_namespace.tsp";
import "./src/routes/v1/app_namespace.tsp";
import "./src/routes/v1/auth.tsp";
import "./src/routes/v1/status.tsp";

using Http;
using TypeSpec.OpenAPI;

@service(#{ title: "www-template API" })
@info(#{ version: "1.0.0" })
@server("/", "Default server")
namespace WWWTemplate;
