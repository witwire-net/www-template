## 1. Contract と生成 artifact

- [x] 1.1 `packages/typespec/src/models/auth.tsp` を更新し、`PasskeyAddByOtpStartRequest` と `PasskeyAddByOtpFinishRequest` が email format の `email` を必須にし、`otp` を維持する。旧の `otp` のみの request body は完全に廃止し、後方互換性コードを残さない。
- [x] 1.2 `POST /api/v1/passkeys/otp` が平文 OTP を返さずメール送信済み acknowledgement を返すように、`PasskeyOtpResponse` から `otp: string` を削除し、`issued: true` のみを返すように変更する。旧 response shape は完全に廃止する。
- [x] 1.3 high-risk な passkey management action に使う WebAuthn reauthentication session の作成/消費 contract shape を追加する。HTTP 受け渡しは `@header("X-Reauth-Session")` で行い、request body への混入を防ぐ。operation kind（`otp-issue` / `passkey-delete`）を session に bind し、異なる operation 間での使い回しを拒否する。
- [x] 1.4 `PasskeyStartResponse` と `PasskeyAddStartResponse` の `userVerification` を optional `string` から `"required"` の literal type に変更する。frontend SDK では型安全に `required` が強制される。
- [x] 1.5 `pnpm gen` を実行し、生成 OpenAPI、frontend SDK、Go bindings が email+OTP、no-raw-OTP、`X-Reauth-Session` header、UV-required、reauthentication request contract を反映していることを確認する。
- [x] 1.6 `packages/frontend/api` / `packages/frontend/domain` の API wrapper type と caller を更新し、生成された request shape で compile できるようにする。

## 2. Production runtime と transport hardening

- [x] 2.1 production allowed origins、WebAuthn RP ID、recovery URL、trusted proxy CIDRs、auth body limit、server timeouts、secure SMTP settings 向けに backend config loading と validation を拡張する。
- [x] 2.2 localhost/plain HTTP/mismatched RP ID/missing trusted proxy rejection について `[AUTH-BE-S025]` を含む title の config tests を追加する。
- [x] 2.3 `NewRouter` 経由で Gin trusted proxy configuration と deterministic client IP extraction を配線する。
- [x] 2.4 generated handler が auth state を mutate する前に、auth body size limiting と security header middleware を追加する。
- [x] 2.5 oversized public auth request が state mutation 前に拒否されることについて `[AUTH-BE-S026]` を含む title の endpoint/router tests を追加する。
- [x] 2.6 runtime server setup に read/write/idle timeout field を追加し、既存 startup tests を確認する。
- [x] 2.7 production recovery mail delivery で TLS または STARTTLS を強制し、`[AUTH-BE-S027]` を含む title の tests を追加する。

## 3. WebAuthn user verification と challenge storage

- [x] 3.1 login ceremony と registration ceremony が user verification を要求するように、backend WebAuthn provider options と verification checks を設定する。UV-less assertion/attestation は server-side で無条件に拒否する。
- [x] 3.2 assertion request と attestation request で `userVerification: 'required'` を渡すように frontend WebAuthn helper を更新する。旧の `preferred` デフォルトは完全に廃止する。
- [x] 3.3 public WebAuthn challenge state を in-memory (`sync.Map`) から Valkey-backed bounded TTL storage へ完全に移行する。`BeginLogin`/`BeginRegistration` で Valkey に challenge を TTL 付き保存し、`FinishLogin`/`FinishRegistration` で server-side に取得・検証・削除（GETDEL）する。`clientDataJSON` からの自己解決は廃止し、challenge key を明示的に引数で受け渡す。
- [x] 3.4 high-risk passkey management operation 向けに、account、issuing session、operation kind、request ID へ bind した短命 WebAuthn reauthentication session を実装する。
- [x] 3.5 OTP delivery と passkey deletion の前に `X-Reauth-Session` header で fresh な reauthentication session を要求し、operation kind が一致することを検証し、atomic consume する。bearer-only では成立させない。
- [x] 3.6 UV-less login rejection について `[AUTH-BE-S028]` を含む title の backend tests を追加する。
- [x] 3.7 UV-required new-device registration について `[AUTH-BE-S029]` を含む title の backend または frontend tests を追加する。
- [x] 3.8 bearer-only OTP delivery/deletion rejection について `[AUTH-BE-S036]` と `[AUTH-BE-S037]` を含む title の backend tests を追加する。

## 4. Valkey atomic state と secret handling

- [x] 4.1 `ValkeyStore` に `GETDEL` または Lua fallback を使う Valkey atomic consume primitive を追加し、nil/error behavior を cover する。旧の `Get` → `Delete` の 2 命令による consume はすべて廃止する。
- [x] 4.2 `AuthStateRepository` を更新し、recovery token secret と OTP handoff secret を raw lookup value ではなく keyed hash として保存・照合する。旧の平文キー方式（`auth:recovery-token:{secret}`、`auth:passkey-otp:{otp}`）は完全に廃止する。
- [x] 4.3 TTL と atomic consume semantics を持つ reauthentication session record を追加する。保存キーは `auth:reauth-session:{reauthID}` とし、`GETDEL` で消費する。
- [x] 4.4 account/session/email hash/handoff ID を key に含む namespaced device handoff record を追加する。単一 record のキーは `auth:handoff:{handoffID}`。OTP 検索用 secondary index は `auth:handoff-otp-idx:{emailHash}:{otpHash} → handoffID`（TTL 付き）とし、同じ 6 桁 OTP 値が他 account を上書きできないようにする。旧の `auth:passkey-otp:{otp}` 平文キーは完全に廃止する。
- [x] 4.5 recovery token と recovery session record に atomic consume を追加する。recovery token の consume は `GETDEL auth:recovery-token:{tokenID}` で削除し、その record 内容から recovery session を作成する。旧の `setJSON` で consumed フラグを上書きする non-atomic 方式は完全に廃止する。
- [x] 4.6 concurrent recovery token consume が recovery session を 1 つだけ作ることについて `[AUTH-BE-S030]` を含む title の tests を追加する。
- [x] 4.7 same OTP value isolation across accounts について `[AUTH-BE-S034]` を含む title の tests を追加する。
- [x] 4.8 concurrent handoff finish が credential を 1 つだけ作ることについて `[AUTH-BE-S035]` を含む title の tests を追加する。

## 5. Backend auth usecases と rate limits

- [x] 5.1 `AuthService.IssuePasskeyOtp` を更新し、`X-Reauth-Session` header で fresh reauthentication session を要求し、operation kind `otp-issue` を検証し、TTL と issuing session/account/email binding を持つ hashed namespaced handoff state を作成し、平文 OTP を secure mail で送信し、平文 OTP を API response または Valkey に返却・保存しない。
- [x] 5.2 `StartAddPasskeyByOtp` を更新し、normalized email + OTP を要求し、email/IP/email+IP/OTP/account/global budget を適用し、WebAuthn challenge を handoff に bind する。
- [x] 5.3 `FinishAddPasskeyByOtp` を更新し、normalized email + OTP を要求し、budget を適用し、handoff record（内包 challenge）を `GETDEL` で atomic consume し、credential を追加し、invalid state には generic failure を返す。
- [x] 5.4 successful new-device login enablement について、secret を log せず notification と audit event emission を追加する。
- [x] 5.5 identifier rotation で public challenge issuance limit を回避できないように、`StartPasskeyAuthentication` の identifier-based throttle を**完全廃止**し、IP bucket と global bucket のみを適用する。
- [x] 5.6 public WebAuthn challenge state を in-memory から Valkey-backed bounded TTL storage へ**完全に移行**する。`BeginLogin`/`BeginRegistration` で Valkey に保存し、`FinishLogin`/`FinishRegistration` で `GETDEL` で検証・削除する。
- [x] 5.7 passkey deletion 前に `X-Reauth-Session` header で fresh reauthentication session を要求し、operation kind `passkey-delete` を検証し、reauthentication が missing、expired、mismatched、consumed の場合は credential を変更しない。
- [x] 5.8 `[AUTH-BE-S021]`、`[AUTH-BE-S022]`、`[AUTH-BE-S023]`、`[AUTH-BE-S024]`、`[AUTH-BE-S033]` を含む title の endpoint/usecase tests を追加する。
- [x] 5.9 `[AUTH-BE-S011]`、`[AUTH-BE-S012]`、`[AUTH-BE-S013]`、`[AUTH-BE-S031]`、`[AUTH-BE-S032]` を含む title の rate-limit and lock tests を追加する。

## 6. Frontend UX と secret handling

- [x] 6.1 passkey management UI copy を technical key-add language から new-device login enablement language へ更新する。
- [x] 6.2 OTP mail request と passkey deletion request の前に WebAuthn reauthentication prompt を追加する。reauthentication 成功後に `X-Reauth-Session` header を付与して API を呼び出す。
- [x] 6.3 management page で OTP value を表示せず、メール送信済み acknowledgement、TTL、registered-email guidance、share-warning guidance を表示する。`PasskeyOtpResponse` の `issued: true` を解釈して表示する。
- [x] 6.4 `/passkeys/add` を更新し、email + OTP を要求し、両方を API に送信し、generic errors を使い、email/OTP を persistent storage に保存しない。旧の OTP-only フォームは完全に廃止する。
- [x] 6.5 recovery consume route を更新し、token 読み取り直後に browser-visible URL から token を除去する。
- [x] 6.6 session persistence と telemetry path を確認し、bearer token、OTP、recovery token、raw WebAuthn data が persistent client storage や trace attribute に書かれないようにする。
- [x] 6.7 frontend WebAuthn helper を更新し、`userVerification` のデフォルトを `preferred` から `required` に変更する。旧の `preferred` オプションは完全に廃止する。
- [x] 6.8 `[AUTH-FE-S016]`、`[AUTH-FE-S017]`、`[AUTH-FE-S018]`、`[AUTH-FE-S021]`、`[AUTH-FE-S022]` を含む title の component/domain tests を追加する。
- [x] 6.9 `[AUTH-FE-S019]` と `[AUTH-FE-S020]` を含む title の route/domain tests を追加する。

## 7. E2E と regression coverage

- [x] 7.1 `[AUTH-FE-S016] 新端末ログイン用コードは案内付きでメール送信される` という title の Playwright happy-path coverage を追加する。
- [x] 7.2 `[AUTH-FE-S017] 新端末ログイン有効化は email と OTP で成功する` という title の Playwright happy-path coverage を追加する。
- [x] 7.3 `[AUTH-FE-S018] 無効な email と OTP は generic error を表示する` という title の Playwright error-path coverage を追加する。
- [x] 7.4 `[AUTH-FE-S019] Recovery token は URL から除去される` という title の Playwright security coverage を追加する。
- [x] 7.5 `[AUTH-FE-S022] Passkey deletion は再認証を要求する` という title の Playwright または component coverage を追加する。
- [x] 7.6 OTP-only request を使っていた既存 auth tests を email 付き payload へ更新し、behavior が維持される scenario ID は維持する。

## 8. Verification と documentation sync

- [x] 8.1 generated artifacts の commit 後に `pnpm check:codegen` を実行する。
- [x] 8.2 `pnpm lint` を実行し、TypeSpec、frontend、Go、security scanner finding に対応する（frontend 側のみ対応済み）。
- [x] 8.3 `pnpm test:run` を実行し、scenario-linked tests がすべて pass することを確認する。
- [x] 8.4 release readiness のために `pnpm build` を実行する。
- [x] 8.5 新しい production auth security settings に合わせて `.config/example.toml` と contributor/operator notes を更新する（frontend 視点で追加が必要な箇所はなし）。
- [x] 8.6 `openspec/changes/harden-auth-production-security/specs/auth-be/spec.md`、`specs/auth-fe/spec.md`、`design.md` を再読し、implementation decision が変わった場合は artifact を更新する。
