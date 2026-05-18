import { json } from '@sveltejs/kit';

import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  fail,
  NO_STORE_HEADERS,
  requireAuthenticatedOperator,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 認証済みオペレーターの passkey 削除 API。
 *
 * @param event SvelteKit request event
 * @returns 削除結果
 */
export const DELETE: RequestHandler = async (event) => {
  // URL パラメータの passkey は必ずログイン済み operator の所有物として検索する。
  const operator = requireAuthenticatedOperator(event);
  const passkeyId = event.params.id;
  if (passkeyId === undefined || passkeyId === '') fail(400, 'Invalid passkey id');
  await getAdminPrisma().$transaction(
    async (tx) => {
      const passkey = await passkeyModel.findOperatorPasskeyForOperator(tx, operator.id, passkeyId);
      if (passkey === null) fail(403, 'Forbidden');
      // 最後の passkey を削除するとロックアウトするため拒否する。
      const count = await passkeyModel.getPasskeyCount(tx, operator.id);
      if (count <= 1) fail(400, 'Cannot delete the last passkey');
      await passkeyModel.deleteOperatorPasskey(tx, passkey.id);
    },
    { isolationLevel: 'Serializable' }
  );
  return json({ deleted: true }, { headers: NO_STORE_HEADERS });
};
