import "@typespec/http";

using Http;

@useAuth(BearerAuth)
@route("/api/v1/app")
namespace WwwTemplate.ApiV1App;
