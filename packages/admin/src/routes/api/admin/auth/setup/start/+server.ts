import { json } from '@sveltejs/kit';

import { generateRegistrationChallenge } from '$lib/server/infrastructure/auth/registration.js';
import { getAdminBootstrapConfig } from '$lib/server/infrastructure/config/env.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import {
  enforcePreAuthRateLimit,
  fail,
  NO_STORE_HEADERS,
  parseJson,
  requireValkey,
  setupStartRequestSchema,
  sha256,
  verifyBootstrapSecret,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 初回 admin セットアップ開始 API。
 *
 * @param event SvelteKit request event
 * @returns WebAuthn 登録 challenge response
 */
export const POST: RequestHandler = async (event) => {
  // Valkey は brute-force 防御境界なので、secret 検証より前に fail-close で確認する。
  const valkey = await requireValkey();
  const input = await parseJson(event, setupStartRequestSchema);
  await enforcePreAuthRateLimit(event, 'bootstrap', sha256(input.bootstrapSecret), valkey);
  // 既存 operator がある環境では初回セットアップ route を使用できない。
  if ((await operatorModel.countOperators(getAdminPrisma())) !== 0)
    fail(409, 'Bootstrap is not allowed');
  const env = getAdminBootstrapConfig();
  if (!env.adminBootstrapEnabled) fail(403, 'Bootstrap is disabled');
  if (env.adminBootstrapExpiresAt < new Date()) fail(403, 'Bootstrap has expired');
  if (!verifyBootstrapSecret(input.bootstrapSecret)) fail(403, 'Invalid bootstrap secret');
  // challenge には作成予定 operator 情報を保存し、finish で再確認後に DB 作成する。
  const challenge = await generateRegistrationChallenge(
    {
      type: 'setup',
      operatorId: `bootstrap:${sha256(input.email)}`,
      email: input.email.trim().toLowerCase(),
      displayName: input.displayName.trim(),
      excludeCredentialIds: [],
    },
    valkey
  );
  return json(
    { challengeId: challenge.challengeId, options: challenge.options },
    { headers: NO_STORE_HEADERS }
  );
};
