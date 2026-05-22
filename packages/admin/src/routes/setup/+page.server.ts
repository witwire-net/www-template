import { error, redirect } from '@sveltejs/kit';

import { getAdminBootstrapConfig } from '$lib/server/infrastructure/config/env.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * 初回 Admin セットアップページの server load。
 *
 * admin.operators が 0 件の環境だけ bootstrap UI を表示し、既に初期化済みなら login へ戻す。
 */
export const load: ServerLoad = async () => {
  const { adminBootstrapEnabled, adminBootstrapExpiresAt } = getAdminBootstrapConfig();
  if (!adminBootstrapEnabled || adminBootstrapExpiresAt.getTime() <= Date.now()) {
    error(403, 'Admin bootstrap is not available');
  }

  if ((await operatorModel.countOperators(getAdminPrisma())) !== 0) {
    redirect(303, '/login');
  }

  return { bootstrapAllowed: true };
};
