import { json } from '@sveltejs/kit';

import { generateRegistrationChallenge } from '$lib/server/infrastructure/auth/registration.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  enforcePreAuthRateLimit,
  fail,
  findOperatorBySetupToken,
  NO_STORE_HEADERS,
  operatorSetupStartRequestSchema,
  parseJson,
  requireValkey,
  sha256,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 追加オペレーターの passkey セットアップ開始 API。
 *
 * @param event SvelteKit request event
 * @returns WebAuthn 登録 challenge response
 */
export const POST: RequestHandler = async (event) => {
  // Valkey rate-limit が使えない場合は token 検証前に fail-close する。
  const valkey = await requireValkey();
  const input = await parseJson(event, operatorSetupStartRequestSchema);
  const fingerprint = sha256(input.setupToken);
  await enforcePreAuthRateLimit(event, 'operator-setup', fingerprint, valkey);
  // bcrypt token 検証は期限内・active operator に限定して行う。
  const operator = await findOperatorBySetupToken(input.setupToken);
  if (operator === null) fail(401, 'Setup token is invalid');
  const passkeys = await passkeyModel.listOperatorPasskeys(getAdminPrisma(), operator.id);
  if (passkeys.length > 0) fail(401, 'Setup token is invalid');
  const challenge = await generateRegistrationChallenge(
    {
      type: 'operator-setup',
      operatorId: operator.id,
      email: operator.email,
      displayName: operator.displayName,
      excludeCredentialIds: passkeys.map((passkey) => passkey.credentialHandle),
    },
    valkey
  );
  return json(
    { challengeId: challenge.challengeId, options: challenge.options },
    { headers: NO_STORE_HEADERS }
  );
};
