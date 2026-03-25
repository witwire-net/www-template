import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("Canonical ULID string used for auth-owned resource and correlation identifiers.")
scalar UlidId extends string;

enum AuthFailureClassification {
  unauthenticated: "unauthenticated",
  sessionExpired: "session-expired",
  internalError: "internal-error",
}

model AuthFailureResponse {
  requestId: UlidId;
  error: AuthFailureClassification;
}

model AuthOperationErrorResponse {
  requestId: UlidId;
  error: string;
}

model PasskeyStartRequest {
  identifier: string;
}

model PasskeyStartResponse {
  requestId: UlidId;
  challenge: string;
  rpId: string;
}

model PasskeyFinishRequest {
  credential: string;
}

model AuthSessionResponse {
  requestId: UlidId;
  accountId: UlidId;
  passkeyCredentialId: UlidId;
  sessionId: UlidId;
  sessionToken: string;
  expiresAt: utcDateTime;
}

model RecoveryRequest {
  @format("email")
  email: string;
}

model RecoveryAcceptedResponse {
  requestId: UlidId;
  accepted: true;
}

model RecoveryConsumeRequest {
  token: string;
}

model RecoveryConsumeResponse {
  requestId: UlidId;
  recoveryTokenId: UlidId;
  recoverySessionId: UlidId;
  recovery_session: UlidId;
  expiresAt: utcDateTime;
}

model RecoveryPasskeyRegisterRequest {
  recovery_session: UlidId;
  credential: string;
}

model InvitationPasskeyRegisterRequest {
  invitation_session: UlidId;
  credential: string;
}

@oneOf
union PasskeyRegisterRequest {
  recovery: RecoveryPasskeyRegisterRequest,
  invitation: InvitationPasskeyRegisterRequest,
}

model LogoutResponse {
  requestId: UlidId;
  revoked: true;
}
