import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace Admin.ApiV1 {
  @useAuth(WWWTemplate.BearerAuth)
  @tag("admin-accounts")
  @summary("Lists customer accounts for Admin Console operators")
  @operationId("listAdminAccounts")
  @get
  @route("/accounts")
  op listAdminAccounts(@query email?: string, @query cursor?: string, @query limit?: int32):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAccountListResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
      }
    | {
        @statusCode statusCode: 401;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 403;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 503;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      };

  @useAuth(WWWTemplate.BearerAuth)
  @tag("admin-accounts")
  @summary("Creates a customer account through the Admin Console")
  @operationId("createAdminAccount")
  @post
  @route("/accounts")
  op createAdminAccount(
    @header("X-CSRF-Token") csrfToken: string,
    @body body: Admin.AdminCreateAccountRequest,
  ):
    | {
        @statusCode statusCode: 201;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminCreateAccountResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
      }
    | {
        @statusCode statusCode: 401;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 403;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 409;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
      }
    | {
        @statusCode statusCode: 503;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      };

  @useAuth(WWWTemplate.BearerAuth)
  @tag("admin-accounts")
  @summary("Gets one customer account for the Admin Console")
  @operationId("getAdminAccount")
  @get
  @route("/accounts/{accountId}")
  op getAdminAccount(@path accountId: WWWTemplate.UlidId):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAccountDetailResponse;
      }
    | {
        @statusCode statusCode: 401;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 403;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 404;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
      }
    | {
        @statusCode statusCode: 503;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      };
}
