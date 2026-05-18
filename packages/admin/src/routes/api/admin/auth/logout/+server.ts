import { redirect } from '@sveltejs/kit';

import { revokeOperatorSession } from '$lib/server/infrastructure/auth/operator';
import { requireValkey } from '$lib/server/services/auth/routes';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * Admin オペレーターのログアウト BFF endpoint。
 *
 * @param event SvelteKit が渡す cookie と認証済み operator locals を含む request event
 * @returns 共有 Valkey infrastructure の Admin 用 logical DB にある active session を失効し、認証 cookie を削除したうえで `/login` へ遷移する redirect response
 */
export const POST: RequestHandler = async ({ cookies, locals }) => {
  // 現在 session を Admin 用 logical DB から削除し、盗まれた cookie の再利用を即時に止める。
  if (locals.operator !== null) {
    const valkey = await requireValkey();
    await revokeOperatorSession(locals.operator.sessionId, valkey);
  }

  // session cookie と CSRF cookie を同時に消し、logout 後の mutation を確実に拒否する。
  cookies.delete('admin_session', { path: '/' });
  cookies.delete('admin_csrf', { path: '/' });
  return redirect(303, '/login');
};
