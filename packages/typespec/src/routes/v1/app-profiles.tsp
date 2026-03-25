import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate.ApiV1App;

@tag("app-profiles")
@summary("List authenticated sample profiles")
@operationId("listAppProfiles")
@get
@route("/profiles")
op listAppProfiles(): {
  @statusCode statusCode: 200;
  @body body: WWWTemplate.Profile[];
} | {
  @statusCode statusCode: 401;
  @body body: WWWTemplate.ErrorResponse;
};

@tag("app-profiles")
@summary("Get an authenticated sample profile by id")
@operationId("getAppProfile")
@get
@route("/profiles/{id}")
op getAppProfile(@path id: int64): {
  @statusCode statusCode: 200;
  @body body: WWWTemplate.Profile;
} | {
  @statusCode statusCode: 401;
  @body body: WWWTemplate.ErrorResponse;
} | {
  @statusCode statusCode: 404;
  @body body: WWWTemplate.ErrorResponse;
};
