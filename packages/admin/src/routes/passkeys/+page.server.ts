import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  requireAuthenticatedOperator,
  serializePasskey,
} from '$lib/server/services/auth/routes.js';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * 認証済み operator の passkey 管理ページ load。
 *
 * hooks.server.ts の session 検証後に、現在 operator 本人の passkey metadata だけを返す。
 */
export const load: ServerLoad = async (event) => {
  // locals.operator を唯一の認証境界として使い、URL や query で対象 operator を切り替えない。
  const operator = requireAuthenticatedOperator(event);
  const passkeys = await passkeyModel.listOperatorPasskeys(getAdminPrisma(), operator.id);

  // credential handle / public key は返さず、UI に必要な公開 metadata に限定する。
  return {
    operator: { email: operator.email, role: operator.role },
    passkeys: passkeys.map(serializePasskey),
  };
};
