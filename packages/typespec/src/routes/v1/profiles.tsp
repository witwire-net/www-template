import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WwwTemplate.ApiV1;

@tag("profiles")
@summary("List sample profiles")
@operationId("listProfiles")
@get
@route("/profiles")
op listProfiles(): {
  @statusCode statusCode: 200;
  @body body: WwwTemplate.Profile[];
};

@tag("profiles")
@summary("Create a sample profile")
@operationId("createProfile")
@post
@route("/profiles")
op createProfile(@body body: WwwTemplate.CreateProfileInput): {
  @statusCode statusCode: 201;
  @body body: WwwTemplate.Profile;
} | {
  @statusCode statusCode: 400;
  @body body: WwwTemplate.ErrorResponse;
};

@tag("profiles")
@summary("Get a sample profile by id")
@operationId("getProfile")
@get
@route("/profiles/{id}")
op getProfile(@path id: int64): {
  @statusCode statusCode: 200;
  @body body: WwwTemplate.Profile;
} | {
  @statusCode statusCode: 400;
  @body body: WwwTemplate.ErrorResponse;
} | {
  @statusCode statusCode: 404;
  @body body: WwwTemplate.ErrorResponse;
};
