import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

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
  @doc("session credential の発行方式。browser は cookie、external client は bearer を指定する。")
  credentialMode: CredentialMode;

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

  @doc("作成する credential が usernameless login で使える discoverable credential であることを要求する。")
  residentKey: "required";

  @doc("residentKey required と同じ要求を boolean でも伝える WebAuthn 標準 field。")
  requireResidentKey: true;

  userVerification: "required";
  attestation?: string;
}

model PasskeyAddFinishRequest {
  credential: WebAuthnAttestationCredential;
}

@doc("Reauthentication session kind bound to a high-risk operation.")
enum ReauthenticationSessionKind {
  deviceLink: "device-link",
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
