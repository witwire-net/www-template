import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WwwTemplate.ApiV1;

@tag("status")
@summary("Returns sample status")
@operationId("getStatus")
@get
@route("/status")
op getStatus(): {
  @statusCode statusCode: 200;
  @body body: WwwTemplate.StatusResponse;
};
