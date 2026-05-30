## 1. API Contract And Codegen

- [ ] 1.1 Update `packages/typespec/src/models/auth.tsp` with `credentialMode`, Web Cookie session response, Bearer session response, CSRF token, and mode-specific refresh DTOs; done when TypeSpec models express both `web-cookie` and `bearer` flows without implicit token body behavior.
- [ ] 1.2 Update `packages/typespec/src/routes/v1/auth.tsp` so passkey finish, recovery register, refresh, logout, and protected mutation contracts expose the new request/response shapes and `X-CSRF-Token` where needed; done when generated route signatures can represent Cookie and Bearer modes separately.
- [ ] 1.3 Run `pnpm gen`; done when `packages/typespec/openapi/openapi.json`, `packages/backend/internal/generated/openapi/openapi.gen.go`, and `packages/frontend/api/src/generated/client.ts` reflect the TypeSpec changes.
- [ ] 1.4 Run `pnpm check:codegen`; done when generated artifacts have no drift.

## 2. Backend Session And Credential Model

- [ ] 2.1 Update `packages/backend/internal/application/auth_contracts.go` to represent mode-specific session results, CSRF token output, Bearer refresh token output, and session metadata CSRF hash; done when application DTOs no longer force every Product session response to carry `AccessToken`.
- [ ] 2.2 Update `packages/backend/internal/application/token_service.go` to issue access credentials, refresh credentials, and CSRF tokens for Web Cookie mode and body token pairs for Bearer mode; done when issue/refresh callers can request either mode explicitly.
- [ ] 2.3 Update `packages/backend/internal/application/auth_service.go` so passkey finish, recovery register, refresh, and logout propagate `credentialMode` and session credential data correctly; done when no handler needs to infer Web vs Bearer from response body fields.
- [ ] 2.4 Update `packages/backend/internal/adapter/valkey/session_store.go` to persist and load CSRF hash in session metadata; done when sessions without CSRF hash fail closed for Cookie mutations.
- [ ] 2.5 Add or update backend application tests for `[AUTH-BE-S043]`, `[AUTH-BE-S044]`, `[AUTH-BE-S045]`, `[AUTH-BE-S046]`, and `[AUTH-BE-S062]`; done when TokenService issue/refresh/expiry/theft tests cover mode-specific credentials.

## 3. Backend HTTP Boundary

- [ ] 3.1 Update `packages/backend/internal/adapter/http/product/auth.go` to extract exactly one Product credential source from Cookie or Bearer and reject ambiguity; done when middleware binds an authorized session context without passing raw tokens to handlers.
- [ ] 3.2 Add Product Origin validation for unsafe Cookie requests; done when disallowed or missing Origin is rejected before protected state mutation.
- [ ] 3.3 Add Product CSRF validation for Cookie state-changing requests; done when `X-CSRF-Token` is compared against session-bound metadata before handler execution.
- [ ] 3.4 Update `packages/backend/internal/adapter/http/product/router.go` Cookie helpers for `access_token` and `refresh_token`, including Set-Cookie and clear behavior; done when Web Cookie login/refresh/logout set and clear the correct HttpOnly cookies.
- [ ] 3.5 Update Product strict handlers in `router.go` to use mode-specific generated DTOs for passkey finish, register, refresh, logout, passkey management, account settings, and session management; done when handlers no longer read `Authorization` directly for Web Cookie requests.
- [ ] 3.6 Add or update backend endpoint tests for recovery/device-link scenarios `[AUTH-BE-S004]`, `[AUTH-BE-S005]`, `[AUTH-BE-S006]`, `[AUTH-BE-S030]`, `[AUTH-BE-S047]`, and `[AUTH-BE-S060]`; done when token issuance/consume/device-link Cookie session coverage exists.
- [ ] 3.7 Add or update backend endpoint tests for recovery register scenarios `[AUTH-BE-S007]`, `[AUTH-BE-S008]`, and `[AUTH-BE-S048]`; done when register returns mode-specific sessions and preserves recovery/device-link post-processing.
- [ ] 3.8 Add or update backend endpoint tests for passkey management scenarios `[AUTH-BE-S014]`, `[AUTH-BE-S015]`, `[AUTH-BE-S016]`, `[AUTH-BE-S017]`, `[AUTH-BE-S018]`, `[AUTH-BE-S019]`, and `[AUTH-BE-S061]`; done when Cookie/Bearer session sources and ambiguity rejection are covered.
- [ ] 3.9 Add or update backend endpoint tests for WebAuthn reauthentication scenarios `[AUTH-BE-S028]`, `[AUTH-BE-S029]`, `[AUTH-BE-S036]`, and `[AUTH-BE-S037]`; done when high-risk operations require reauthentication with either accepted session credential source.
- [ ] 3.10 Add or update backend endpoint tests for device-link scenarios `[AUTH-BE-S049]` and `[AUTH-BE-S050]`; done when device-link delivery works with active application session plus reauth and fails without reauth.
- [ ] 3.11 Add or update backend endpoint tests for session issuance/authorization scenarios `[AUTH-BE-S001]`, `[AUTH-BE-S002]`, `[AUTH-BE-S003]`, `[AUTH-BE-S009]`, `[AUTH-BE-S010]`, `[AUTH-BE-S054]`, `[AUTH-BE-S055]`, `[AUTH-BE-S058]`, `[AUTH-BE-S063]`, `[AUTH-BE-S064]`, and `[AUTH-BE-S065]`; done when Bearer mode, Cookie mode, logout, missing/expired/suspended, CSRF, and ambiguity behavior are covered.

## 4. Frontend Auth State And API Calls

- [ ] 4.1 Update `packages/frontend/domain/src/auth/types.ts` so Web auth state stores session metadata and CSRF token without `accessToken`; done when TypeScript types make browser-readable Product accessToken unavailable to Web hooks.
- [ ] 4.2 Update `packages/frontend/domain/src/auth/session/state.ts` to replace Authorization header generation with same-origin credential request helpers and CSRF header helpers; done when Product Web domain code has no `Authorization: Bearer` creation path.
- [ ] 4.3 Update `packages/frontend/domain/src/auth/session/hook.svelte.ts` for bootstrap refresh, session-expired refresh-once retry, logout, session clearing, account-suspended routing, and AccountSetting snapshot handling; done when auth state is driven by Cookie responses and CSRF token rotation.
- [ ] 4.4 Update `packages/frontend/domain/src/auth/passkey/login/hook.svelte.ts` to send `credentialMode="web-cookie"` and accept Web Cookie session responses; done when login does not decode or store accessToken.
- [ ] 4.5 Update `packages/frontend/domain/src/auth/recovery/hook.svelte.ts` to send `credentialMode="web-cookie"` for register and accept CSRF/session metadata; done when recovery/device-link registration enters authenticated state without token body.
- [ ] 4.6 Update `packages/frontend/domain/src/auth/passkey/management/hook.svelte.ts` and `packages/frontend/domain/src/auth/session/session_api.ts` to send same-origin credentials plus CSRF headers for mutations; done when passkey/device/session management no longer receives Authorization headers from callers.
- [ ] 4.7 Update `packages/frontend/domain/src/account/hook.svelte.ts` and `packages/frontend/domain/src/account/localeSync.svelte.ts` to use Cookie + CSRF request helpers; done when AccountSetting load/update works without bearer headers.
- [ ] 4.8 Update `packages/frontend/app/src/tests/mocks/handlers.ts` and related app mocks to return Web Cookie mode response bodies; done when frontend tests no longer depend on mocked `accessToken` body for Product Web.

## 5. Frontend Tests

- [ ] 5.1 Add or update frontend tests for login scenarios `[AUTH-FE-S001]`, `[AUTH-FE-S002]`, and `[AUTH-FE-S045]`; done when passkey login enters authenticated state from CSRF/session metadata without accessToken storage.
- [ ] 5.2 Add or update frontend tests for recovery/device-link scenarios `[AUTH-FE-S003]`, `[AUTH-FE-S004]`, `[AUTH-FE-S005]`, `[AUTH-FE-S038]`, and `[AUTH-FE-S046]`; done when recovery registration accepts Web Cookie session responses.
- [ ] 5.3 Add or update frontend tests for refresh/session continuation scenarios `[AUTH-FE-S023]`, `[AUTH-FE-S024]`, `[AUTH-FE-S025]`, `[AUTH-FE-S026]`, and `[AUTH-FE-S047]`; done when bootstrap refresh, retry-on-session-expired, and no persistent token storage are covered.
- [ ] 5.4 Add or update frontend tests for single active Cookie session scenarios `[AUTH-FE-S027]`, `[AUTH-FE-S028]`, `[AUTH-FE-S029]`, `[AUTH-FE-S030]`, and `[AUTH-FE-S031]`; done when account switching token-list behavior is removed from Product Web.
- [ ] 5.5 Add or update frontend tests for expiry/logout routing scenarios `[AUTH-FE-S006]`, `[AUTH-FE-S007]`, and `[AUTH-FE-S008]`; done when missing session, expired session, and logout route intents are distinct.
- [ ] 5.6 Add or update frontend tests for passkey management scenarios `[AUTH-FE-S010]`, `[AUTH-FE-S011]`, `[AUTH-FE-S012]`, `[AUTH-FE-S013]`, `[AUTH-FE-S014]`, `[AUTH-FE-S015]`, `[AUTH-FE-S035]`, and `[AUTH-FE-S037]`; done when management API calls include credentials and CSRF where expected.
- [ ] 5.7 Add or update frontend tests for security presentation scenarios `[AUTH-FE-S019]`, `[AUTH-FE-S020]`, and `[AUTH-FE-S048]`; done when tokens/Cookie values are absent from state/storage and auth routes keep no-store/security behavior.
- [ ] 5.8 Add or update frontend tests for suspended account scenarios `[AUTH-FE-S041]`, `[AUTH-FE-S042]`, `[AUTH-FE-S043]`, and `[AUTH-FE-S044]`; done when suspended handling clears Cookie session state and avoids public enumeration.
- [ ] 5.9 Add or update frontend tests for device management scenarios `[AUTH-FE-S034]`, `[AUTH-FE-S049]`, and `[AUTH-FE-S036]`; done when session list/revoke/revoke-others use Cookie + CSRF requests.

## 6. Verification And Artifact Maintenance

- [ ] 6.1 Run `pnpm lint`; done when lint passes through repository scripts.
- [ ] 6.2 Run `pnpm check`; done when TypeScript/Svelte/Go type checks pass through repository scripts.
- [ ] 6.3 Run `pnpm test:server`; done when backend tests covering `[AUTH-BE-*]` scenarios pass.
- [ ] 6.4 Run `pnpm test:client`; done when frontend tests covering `[AUTH-FE-*]` scenarios pass.
- [ ] 6.5 Run `pnpm check:codegen` after all implementation edits; done when TypeSpec-generated artifacts are in sync.
- [ ] 6.6 Update OpenSpec delta specs/design/tasks if implementation uncovers a contract mismatch; done when artifacts and code agree before archive or sync.
