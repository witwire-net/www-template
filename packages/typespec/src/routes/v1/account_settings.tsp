import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate.ApiV1App {
  @tag("account-settings")
  @summary("Gets the current account settings")
  @operationId("getAccountSettings")
  @get
  @route("/account/settings")
  op getAccountSettings():
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AccountSettingResponse;
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

  @tag("account-settings")
  @summary("Updates the current account settings")
  @operationId("updateAccountSettings")
  @patch
  @route("/account/settings")
  op updateAccountSettings(@body body: WWWTemplate.UpdateAccountSettingRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AccountSettingResponse;
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
}
