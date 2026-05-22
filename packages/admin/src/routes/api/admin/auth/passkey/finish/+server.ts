import { consumeChallenge, verifyAssertion } from '$lib/server/infrastructure/auth/operator.js';
import { getAdminAuthConfig } from '$lib/server/infrastructure/config/env.js';
import { getPlatformConfig } from '$lib/server/infrastructure/config/platform.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  challengeFinishRequestSchema,
  enforcePreAuthRateLimit,
  fail,
  parseJson,
  requireValkey,
  sessionRedirectResponse,
  sha256,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

const AUTHENTICATION_FAILED_MESSAGE = 'Authentication failed';

/**
 * passkey ログイン完了 API。
 *
 * @param event SvelteKit request event
 * @returns 認証成功時は session cookie 付き 303 redirect
 */
export const POST: RequestHandler = async (event) => {
  // 未検証 JSON を route 内で直接使わないため、challengeId と assertion を schema で受ける。
  const input = await parseJson(event, challengeFinishRequestSchema);
  const valkey = await requireValkey();
  await enforcePreAuthRateLimit(event, 'login-finish', sha256(input.challengeId), valkey);
  try {
    // consumeChallenge は GETDEL なので、成功・失敗に関わらず challenge reuse を防ぐ。
    const challenge = await consumeChallenge(input.challengeId, 'login', valkey);
    if (challenge.operatorId.startsWith('decoy:')) fail(401, AUTHENTICATION_FAILED_MESSAGE);
    // challenge に保存された operator と DB 現在状態を照合し、inactive 化された operator を拒否する。
    const operator = await operatorModel.findOperatorById(getAdminPrisma(), challenge.operatorId);
    if (operator?.isActive !== true) fail(401, AUTHENTICATION_FAILED_MESSAGE);
    if (operator.email !== challenge.email) fail(401, AUTHENTICATION_FAILED_MESSAGE);
    // WebAuthn credential ID から所有 passkey を取得し、operator binding を検証する。
    const assertion = input.assertion as { id?: string };
    if (typeof assertion.id !== 'string' || assertion.id === '')
      fail(401, AUTHENTICATION_FAILED_MESSAGE);
    const passkey = await passkeyModel.findOperatorPasskeyByCredentialHandle(
      getAdminPrisma(),
      assertion.id
    );
    if (passkey?.operatorId !== operator.id) fail(401, AUTHENTICATION_FAILED_MESSAGE);
    const { adminOrigin } = getAdminAuthConfig();
    const { adminRpId } = getPlatformConfig();
    // Assertion 検証成功後だけ sign_count と last_login_at を更新する。
    const verified = await verifyAssertion(
      input.assertion as Parameters<typeof verifyAssertion>[0],
      challenge.challenge,
      {
        credential_handle: passkey.credentialHandle,
        public_key: Buffer.from(passkey.publicKey),
        sign_count: passkey.signCount,
        transports: passkey.transports,
      },
      adminOrigin,
      adminRpId
    );
    await passkeyModel.updateOperatorPasskeySignCount(
      getAdminPrisma(),
      passkey.id,
      verified.newSignCount
    );
    await operatorModel.updateLoginTimestamp(getAdminPrisma(), operator.id);
    return await sessionRedirectResponse(operator, valkey);
  } catch {
    // unknown email / invalid assertion / expired challenge は同じ 401 に集約し、列挙を防ぐ。
    fail(401, AUTHENTICATION_FAILED_MESSAGE);
  }
};
