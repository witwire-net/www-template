import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate.ApiV1;

@tag("profiles")
@summary("List sample profiles")
@operationId("listProfiles")
@get
@route("/profiles")
op listProfiles(): {
  @statusCode statusCode: 200;
  @body body: WWWTemplate.Profile[];
};

@tag("profiles")
@summary("Create a sample profile")
@operationId("createProfile")
@post
@route("/profiles")
op createProfile(@body body: WWWTemplate.CreateProfileInput): {
  @statusCode statusCode: 201;
  @body body: WWWTemplate.Profile;
} | {
  @statusCode statusCode: 400;
  @body body: WWWTemplate.ErrorResponse;
};

@tag("profiles")
@summary("Get a sample profile by id")
@operationId("getProfile")
@get
@route("/profiles/{id}")
op getProfile(@path id: int64): {
  @statusCode statusCode: 200;
  @body body: WWWTemplate.Profile;
} | {
  @statusCode statusCode: 400;
  @body body: WWWTemplate.ErrorResponse;
} | {
  @statusCode statusCode: 404;
  @body body: WWWTemplate.ErrorResponse;
};
