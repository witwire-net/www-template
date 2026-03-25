import "@typespec/http";
import "@typespec/openapi";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate.ApiV1 {
  @tag("auth")
  @summary("Starts a passkey authentication ceremony")
  @operationId("startPasskeyAuthentication")
  @post
  @route("/auth/passkey/start")
  op startPasskeyAuthentication(@body body: WWWTemplate.PasskeyStartRequest): {
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

  @tag("auth")
  @summary("Finishes passkey authentication and returns a bearer session")
  @operationId("finishPasskeyAuthentication")
  @post
  @route("/auth/passkey/finish")
  op finishPasskeyAuthentication(@body body: WWWTemplate.PasskeyFinishRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthSessionResponse;
  } | {
    @statusCode statusCode: 400;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthOperationErrorResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("auth")
  @summary("Accepts a recovery request without revealing account existence")
  @operationId("requestPasskeyRecovery")
  @post
  @route("/auth/recovery")
  op requestPasskeyRecovery(@body body: WWWTemplate.RecoveryRequest): {
    @statusCode statusCode: 202;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.RecoveryAcceptedResponse;
  } | {
    @statusCode statusCode: 400;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthOperationErrorResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("auth")
  @summary("Consumes a recovery token and issues a recovery session")
  @operationId("consumeRecoveryToken")
  @post
  @route("/auth/recovery/consume")
  op consumeRecoveryToken(@body body: WWWTemplate.RecoveryConsumeRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.RecoveryConsumeResponse;
  } | {
    @statusCode statusCode: 400;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthOperationErrorResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("auth")
  @summary("Registers or re-registers a passkey from exactly one auth selector")
  @operationId("registerPasskey")
  @post
  @route("/auth/passkey/register")
  op registerPasskey(@body body: WWWTemplate.PasskeyRegisterRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthSessionResponse;
  } | {
    @statusCode statusCode: 400;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthOperationErrorResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };
}

namespace WWWTemplate.ApiV1App {
  @tag("app-auth")
  @summary("Revokes the current bearer session")
  @operationId("logout")
  @post
  @route("/auth/logout")
  op logout(): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.LogoutResponse;
  } | {
    @statusCode statusCode: 401;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };
}
