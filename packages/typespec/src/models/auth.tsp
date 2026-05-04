import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("JWT 形式のアクセストークンを使用した Bearer 認証。Authorization ヘッダーに `Bearer <JWT>` を付与すること。トークンは短命の JWT であり、accountID・sessionID・iat・exp を含む。")
model BearerAuth {
  @doc("Http authentication")
  type: AuthType.http;

  @doc("bearer auth scheme")
  scheme: "bearer";
}

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

@doc("WebAuthn PublicKeyCredentialRequestOptions returned by the server after BeginLogin.")
model PasskeyStartResponse {
  requestId: UlidId;

  @doc("base64url-encoded challenge bytes issued by the server.")
  challenge: string;

  rpId: string;

  @doc("Timeout in milliseconds suggested by the server.")
  timeout?: int64;

  @doc("Allowed credential descriptors (empty for usernameless flows).")
  allowCredentials?: WebAuthnCredentialDescriptor[];

  userVerification: "required";
}

@doc("Credential descriptor used in allowCredentials and excludeCredentials lists.")
model WebAuthnCredentialDescriptor {
  type: string;

  @doc("base64url-encoded credential ID.")
  id: string;

  transports?: string[];
}

@doc("WebAuthn user entity (id is base64url-encoded bytes).")
model WebAuthnUserEntity {
  @doc("base64url-encoded user ID bytes.")
  id: string;

  name: string;
  displayName: string;
}

@doc("WebAuthn public key credential parameter (algorithm descriptor).")
model WebAuthnCredentialParameter {
  type: string;

  @doc("COSE algorithm identifier.")
  alg: int32;
}

@doc("WebAuthn AuthenticatorResponse inner object.")
model WebAuthnAuthenticatorResponse {
  @doc("base64url-encoded clientDataJSON.")
  clientDataJSON: string;
}

@doc("WebAuthn AuthenticatorAssertionResponse (login).")
model WebAuthnAssertionResponse {
  @doc("base64url-encoded clientDataJSON.")
  clientDataJSON: string;

  @doc("base64url-encoded authenticatorData.")
  authenticatorData: string;

  @doc("base64url-encoded signature.")
  signature: string;

  @doc("base64url-encoded userHandle (optional).")
  userHandle?: string;
}

@doc("WebAuthn AuthenticatorAttestationResponse (registration).")
model WebAuthnAttestationResponse {
  @doc("base64url-encoded clientDataJSON.")
  clientDataJSON: string;

  @doc("base64url-encoded attestationObject.")
  attestationObject: string;

  transports?: string[];
}

@doc("WebAuthn PublicKeyCredential for login (navigator.credentials.get result).")
model WebAuthnAssertionCredential {
  id: string;
  rawId: string;
  type: string;
  response: WebAuthnAssertionResponse;
  authenticatorAttachment?: string;
}

@doc("WebAuthn PublicKeyCredential for registration (navigator.credentials.create result).")
model WebAuthnAttestationCredential {
  id: string;
  rawId: string;
  type: string;
  response: WebAuthnAttestationResponse;
  authenticatorAttachment?: string;
}

model PasskeyFinishRequest {
  credential: WebAuthnAssertionCredential;
}

model PasskeyItem {
  id: UlidId;
  identifier: string;
  createdAt: utcDateTime;
}

model PasskeyListResponse {
  requestId: UlidId;
  passkeys: PasskeyItem[];
}

@doc("WebAuthn PublicKeyCredentialCreationOptions returned by the server after BeginRegistration.")
model PasskeyAddStartResponse {
  requestId: UlidId;

  @doc("base64url-encoded challenge bytes issued by the server.")
  challenge: string;

  rpId: string;

  @doc("Relying Party display name.")
  rpName: string;

  @doc("User entity (id is base64url-encoded bytes).")
  user: WebAuthnUserEntity;

  @doc("Supported public key credential algorithms.")
  pubKeyCredParams: WebAuthnCredentialParameter[];

  @doc("Timeout in milliseconds suggested by the server.")
  timeout?: int64;

  excludeCredentials?: WebAuthnCredentialDescriptor[];
  userVerification: "required";
  attestation?: string;
}

model PasskeyAddFinishRequest {
  credential: WebAuthnAttestationCredential;
}

model PasskeyOtpResponse {
  requestId: UlidId;
  issued: true;
}

model PasskeyAddByOtpStartRequest {
  @format("email")
  email: string;

  otp: string;
}

model PasskeyAddByOtpFinishRequest {
  @format("email")
  email: string;

  otp: string;
  credential: WebAuthnAttestationCredential;
}

model AuthSessionResponse {
  requestId: UlidId;
  accountId: UlidId;
  passkeyCredentialId: UlidId;
  sessionId: UlidId;
  accessToken: string;
  refreshToken: string;
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

  @doc("Alias of recoverySessionId for client convenience.")
  recovery_session: UlidId;

  expiresAt: utcDateTime;
}

@doc("Start passkey registration ceremony using a recovery session.")
model RecoveryPasskeyStartRequest {
  recovery_session: UlidId;
}

@doc("Finish passkey registration ceremony using a recovery session.")
model RecoveryPasskeyFinishRequest {
  recovery_session: UlidId;
  credential: WebAuthnAttestationCredential;
}

@doc("Start passkey registration ceremony using an invitation session.")
model InvitationPasskeyStartRequest {
  invitation_session: UlidId;
}

@doc("Finish passkey registration ceremony using an invitation session.")
model InvitationPasskeyFinishRequest {
  invitation_session: UlidId;
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

@doc("Reauthentication session kind bound to a high-risk operation.")
enum ReauthenticationSessionKind {
  otpIssue: "otp-issue",
  passkeyDelete: "passkey-delete",
}

@doc("Reauthentication start request that binds the ceremony to a high-risk operation kind.")
model ReauthenticationStartRequest {
  kind: ReauthenticationSessionKind;
}

@doc("Reauthentication finish request that completes the WebAuthn ceremony and binds to the operation kind.")
model ReauthenticationFinishRequest {
  requestId: UlidId;
  kind: ReauthenticationSessionKind;
  credential: WebAuthnAssertionCredential;
}

@doc("Reauthentication session response issued after a fresh WebAuthn reauthentication.")
model ReauthenticationSessionResponse {
  requestId: UlidId;
  reauthSessionId: UlidId;
  kind: ReauthenticationSessionKind;
  expiresAt: utcDateTime;
}

model LogoutResponse {
  requestId: UlidId;
  revoked: true;
}

@doc("リフレッシュトークンを用いて新しいアクセストークンとリフレッシュトークンのペアを取得するリクエスト。")
model RefreshTokenRequest {
  refreshToken: string;
}

@doc("リフレッシュ成功時に返却される新しいトークンペア。accessToken は短命の JWT、refreshToken は長寿命のローテーション可能トークン。")
model RefreshTokenResponse {
  accessToken: string;
  refreshToken: string;
}

@doc("ログイン中のセッション（デバイス）を表すアイテム。各セッションは一意の sessionId を持つ。")
model SessionItem {
  sessionId: UlidId;
  deviceName: string;
  loginAt: utcDateTime;
  lastActiveAt: utcDateTime;
  ipHash: string;
  isCurrentSession: boolean;
}

@doc("ログイン中のセッション一覧レスポンス。認証済みアカウントが所有する全セッションを含む。")
model SessionListResponse {
  requestId: UlidId;
  sessions: SessionItem[];
}
