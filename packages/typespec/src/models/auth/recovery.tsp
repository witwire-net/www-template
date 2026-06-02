import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

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

@doc("パスキー追加トークンの発行種別。recovery は紛失時復旧、device-link は認証済み端末からの新端末追加。")
enum TokenKind {
  recovery: "recovery",
  deviceLink: "device-link",
}

model RecoveryConsumeResponse {
  requestId: UlidId;
  recoveryTokenId: UlidId;
  recoverySessionId: UlidId;

  @doc("Alias of recoverySessionId for client convenience.")
  recovery_session: UlidId;

  expiresAt: utcDateTime;
  kind: TokenKind;
}

model DeviceLinkResponse {
  requestId: UlidId;
  issued: true;
}

@doc("Start passkey registration ceremony using a recovery session.")
model RecoveryPasskeyStartRequest {
  recovery_session: UlidId;
}

@doc("Finish passkey registration ceremony using a recovery session.")
model RecoveryPasskeyFinishRequest {
  recovery_session: UlidId;

  @doc("session credential の発行方式。browser は cookie、external client は bearer を指定する。")
  credentialMode: CredentialMode;

  credential: WebAuthnAttestationCredential;
}

@doc("Start passkey registration ceremony using an invitation session.")
model InvitationPasskeyStartRequest {
  invitation_session: UlidId;
}

@doc("Finish passkey registration ceremony using an invitation session.")
model InvitationPasskeyFinishRequest {
  invitation_session: UlidId;

  @doc("session credential の発行方式。browser は cookie、external client は bearer を指定する。")
  credentialMode: CredentialMode;

  credential: WebAuthnAttestationCredential;
}

@doc("Start request for passkey registration via recovery session.")
model PasskeyRegisterStartRequest {
  recovery_session: UlidId;
}

@doc("Finish request for passkey register — exactly one selector must be present.")
@oneOf
union PasskeyRegisterRequest {
  recovery: RecoveryPasskeyFinishRequest,
  invitation: InvitationPasskeyFinishRequest,
}
