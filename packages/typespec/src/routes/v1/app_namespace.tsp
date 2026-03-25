import "@typespec/http";

using Http;

@useAuth(BearerAuth)
@route("/api/v1/app")
namespace WWWTemplate.ApiV1App;
