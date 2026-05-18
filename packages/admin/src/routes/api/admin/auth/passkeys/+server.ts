import {
  passkeyListResponse,
  requireAuthenticatedOperator,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 認証済みオペレーターの passkey 一覧 API。
 *
 * @param event SvelteKit request event
 * @returns passkey metadata 一覧
 */
export const GET: RequestHandler = async (event) => {
  // route-level で admin_session を必須にし、未認証時は hooks 未実装でも 401 にする。
  const operator = requireAuthenticatedOperator(event);
  return passkeyListResponse(operator.id);
};
