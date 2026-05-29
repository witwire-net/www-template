import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace Admin.ApiV1 {
  @tag("admin-auth")
  @summary("Starts an Admin operator passkey authentication ceremony")
  @operationId("startAdminPasskeyAuthentication")
  @post
  @route("/auth/passkey/start")
  op startAdminPasskeyAuthentication(@body body: Admin.AdminPasskeyStartRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.PasskeyStartResponse;
  } | {
    @statusCode statusCode: 400;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthOperationErrorResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("admin-auth")
  @summary("Finishes Admin operator passkey authentication and returns an operator session")
  @operationId("finishAdminPasskeyAuthentication")
  @post
  @route("/auth/passkey/finish")
  op finishAdminPasskeyAuthentication(@body body: Admin.AdminPasskeyFinishRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAuthSessionResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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

  @tag("admin-auth")
  @summary("Starts initial Admin operator setup passkey registration")
  @operationId("startAdminInitialSetup")
  @post
  @route("/auth/setup/start")
  op startAdminInitialSetup(@body body: Admin.AdminInitialSetupStartRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.PasskeyAddStartResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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

  @tag("admin-auth")
  @summary("Finishes initial Admin operator setup and returns an operator session")
  @operationId("finishAdminInitialSetup")
  @post
  @route("/auth/setup/finish")
  op finishAdminInitialSetup(@body body: Admin.AdminInitialSetupFinishRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAuthSessionResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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

  @tag("admin-auth")
  @summary("Starts Admin operator setup passkey registration")
  @operationId("startAdminOperatorSetup")
  @post
  @route("/auth/operator-setup/start")
  op startAdminOperatorSetup(@body body: Admin.AdminOperatorSetupStartRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.PasskeyAddStartResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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

  @tag("admin-auth")
  @summary("Finishes Admin operator setup and returns an operator session")
  @operationId("finishAdminOperatorSetup")
  @post
  @route("/auth/operator-setup/finish")
  op finishAdminOperatorSetup(@body body: Admin.AdminOperatorSetupFinishRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAuthSessionResponse;
      }
    | {
        @statusCode statusCode: 400;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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
  @tag("admin-auth")
  @summary("Creates an Admin operator and delivers a one-time setup token")
  @operationId("createAdminOperator")
  @post
  @route("/auth/operators")
  op createAdminOperator(
    @header("X-CSRF-Token") csrfToken: string,
    @body body: Admin.AdminCreateOperatorRequest,
  ):
    | {
        @statusCode statusCode: 201;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminCreateOperatorResponse;
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
  @tag("admin-auth")
  @summary("Gets the current Admin operator")
  @operationId("getCurrentAdminOperator")
  @get
  @route("/auth/operator/current")
  op getCurrentAdminOperator():
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminCurrentOperatorResponse;
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

  @tag("admin-auth")
  @summary("Refreshes the Admin operator access token through an HttpOnly refresh cookie")
  @operationId("refreshAdminOperatorSession")
  @post
  @route("/auth/operator/refresh")
  op refreshAdminOperatorSession():
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminAuthSessionResponse;
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
  @tag("admin-auth")
  @summary("Revokes the current Admin operator session")
  @operationId("logoutAdminOperator")
  @post
  @route("/auth/operator/logout")
  op logoutAdminOperator(@header("X-CSRF-Token") csrfToken: string):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.LogoutResponse;
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
  @tag("admin-auth")
  @summary("Lists registered passkeys for the current Admin operator")
  @operationId("listAdminOperatorPasskeys")
  @get
  @route("/auth/passkeys")
  op listAdminOperatorPasskeys(@header("X-CSRF-Token") csrfToken: string):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: Admin.AdminOperatorPasskeyListResponse;
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
  @tag("admin-auth")
  @summary("Deletes a registered passkey for the current Admin operator")
  @operationId("deleteAdminOperatorPasskey")
  @delete
  @route("/auth/passkeys/{id}")
  op deleteAdminOperatorPasskey(
    @path id: WWWTemplate.UlidId,
    @header("X-CSRF-Token") csrfToken: string,
  ):
    | {
        @statusCode statusCode: 204;
        @header("Cache-Control") cacheControl: "no-store";
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
}
