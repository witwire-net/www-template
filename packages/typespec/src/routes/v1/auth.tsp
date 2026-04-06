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

  @tag("auth")
  @summary("Starts adding a passkey on a new device by OTP handoff")
  @operationId("startPasskeyAdditionByOtp")
  @post
  @route("/auth/passkey/add/start")
  op startPasskeyAdditionByOtp(@body body: WWWTemplate.PasskeyAddByOtpStartRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.PasskeyAddStartResponse;
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
  @summary("Finishes adding a passkey on a new device by OTP handoff")
  @operationId("finishPasskeyAdditionByOtp")
  @post
  @route("/auth/passkey/add/finish")
  op finishPasskeyAdditionByOtp(@body body: WWWTemplate.PasskeyAddByOtpFinishRequest): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
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

  @tag("app-auth")
  @summary("Lists registered passkeys for the current account")
  @operationId("listPasskeys")
  @get
  @route("/passkeys")
  op listPasskeys(): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.PasskeyListResponse;
  } | {
    @statusCode statusCode: 401;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("app-auth")
  @summary("Starts adding a passkey for the current account")
  @operationId("startPasskeyAddition")
  @post
  @route("/passkeys/start")
  op startPasskeyAddition(): {
    @statusCode statusCode: 200;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.PasskeyAddStartResponse;
  } | {
    @statusCode statusCode: 401;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  } | {
    @statusCode statusCode: 503;
    @header("Cache-Control") cacheControl: "no-store";
    @body body: WWWTemplate.AuthFailureResponse;
  };

  @tag("app-auth")
  @summary("Finishes adding a passkey for the current account")
  @operationId("finishPasskeyAddition")
  @post
  @route("/passkeys/finish")
  op finishPasskeyAddition(@body body: WWWTemplate.PasskeyAddFinishRequest):
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.PasskeyListResponse;
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
        @statusCode statusCode: 503;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      };

  @tag("app-auth")
  @summary("Deletes a registered passkey for the current account")
  @operationId("deletePasskey")
  @delete
  @route("/passkeys/{id}")
  op deletePasskey(@path id: WWWTemplate.UlidId):
    | {
        @statusCode statusCode: 204;
        @header("Cache-Control") cacheControl: "no-store";
      }
    | {
        @statusCode statusCode: 401;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      }
    | {
        @statusCode statusCode: 403;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthOperationErrorResponse;
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

  @tag("app-auth")
  @summary("Issues a one-time password for adding a passkey on a new device")
  @operationId("issuePasskeyOtp")
  @post
  @route("/passkeys/otp")
  op issuePasskeyOtp():
    | {
        @statusCode statusCode: 200;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.PasskeyOtpResponse;
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
        @statusCode statusCode: 503;
        @header("Cache-Control") cacheControl: "no-store";
        @body body: WWWTemplate.AuthFailureResponse;
      };
}
