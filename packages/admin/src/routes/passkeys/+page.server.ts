import { issueCsrfToken } from '$lib/server/infrastructure/csrf/guard.js';
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
  const operator = requireAuthenticatedOperator(event);
  const passkeys = await passkeyModel.listOperatorPasskeys(getAdminPrisma(), operator.id);

  return {
    operator: { email: operator.email, role: operator.role },
    passkeys: passkeys.map(serializePasskey),
    csrfToken: issueCsrfToken(operator.sessionId, operator.jti).token,
  };
};
