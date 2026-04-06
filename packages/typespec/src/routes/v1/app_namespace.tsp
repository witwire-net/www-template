import "@typespec/http";

using Http;

@useAuth(BearerAuth)
@route("/api/v1")
namespace WWWTemplate.ApiV1App;
