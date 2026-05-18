import { json } from '@sveltejs/kit';

import { generateRegistrationChallenge } from '$lib/server/infrastructure/auth/registration.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  NO_STORE_HEADERS,
  requireAuthenticatedOperator,
  requireValkey,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 認証済みオペレーターの passkey 追加開始 API。
 *
 * @param event SvelteKit request event
 * @returns WebAuthn 登録 challenge response
 */
export const POST: RequestHandler = async (event) => {
  // ログイン済み operator の DB 現在状態を取得し、inactive 化後の追加を拒否する。
  const localOperator = requireAuthenticatedOperator(event);
  const operator = await operatorModel.findOperatorById(getAdminPrisma(), localOperator.id);
  if (operator?.isActive !== true)
    return new Response('Unauthorized', { status: 401, headers: NO_STORE_HEADERS });
  // 既存 credential を excludeCredentials に渡し、同一 passkey の再登録を避ける。
  const passkeys = await passkeyModel.listOperatorPasskeys(getAdminPrisma(), operator.id);
  const valkey = await requireValkey();
  const challenge = await generateRegistrationChallenge(
    {
      type: 'passkey-add',
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
