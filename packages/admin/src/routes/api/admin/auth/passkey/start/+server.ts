import { json } from '@sveltejs/kit';

import { generateChallenge } from '$lib/server/infrastructure/auth/operator.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  loginStartRequestSchema,
  NO_STORE_HEADERS,
  parseJson,
  requireValkey,
  sha256,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * passkey ログイン開始 API。
 *
 * @param event SvelteKit request event
 * @returns real / decoy を同一 shape にした challenge response
 */
export const POST: RequestHandler = async (event) => {
  // 入力 email は正規化してから検索し、DB の有無による response 差分を作らない。
  const input = await parseJson(event, loginStartRequestSchema);
  const email = input.email.trim().toLowerCase();
  // challenge 保存に必要な Admin Valkey は先に fail-close で確認する。
  const valkey = await requireValkey();
  // active かつ passkey 登録済みの場合だけ real operator binding を作る。
  const operator = await operatorModel.findOperatorByEmail(getAdminPrisma(), email);
  const passkeyCount =
    operator === null ? 0 : await passkeyModel.getPasskeyCount(getAdminPrisma(), operator.id);
  const isReal = operator !== null && operator.isActive && passkeyCount > 0;
  // unknown / inactive / unregistered は decoy operatorId を保存し、finish で同じ 401 に集約する。
  const challenge = await generateChallenge(
    {
      type: 'login',
      operatorId: isReal ? operator.id : `decoy:${sha256(email)}`,
      email,
    },
    valkey
  );
  return json(
    { challengeId: challenge.challengeId, options: challenge.options },
    { headers: NO_STORE_HEADERS }
  );
};
