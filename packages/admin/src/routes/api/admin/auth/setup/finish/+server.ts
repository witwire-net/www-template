import { consumeChallenge } from '$lib/server/infrastructure/auth/operator.js';
import { verifyAttestation } from '$lib/server/infrastructure/auth/registration.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  fail,
  parseJson,
  registrationFinishRequestSchema,
  requireValkey,
  sessionRedirectResponse,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 初回 admin セットアップ完了 API。
 *
 * @param event SvelteKit request event
 * @returns session cookie 付き 303 redirect
 */
export const POST: RequestHandler = async (event) => {
  // challenge は一度だけ消費し、replay による二重作成を防止する。
  const input = await parseJson(event, registrationFinishRequestSchema);
  const valkey = await requireValkey();
  const challenge = await consumeChallenge(input.challengeId, 'setup', valkey).catch(() => {
    fail(401, 'Setup failed');
  });
  if (challenge.displayName === undefined || !challenge.operatorId.startsWith('bootstrap:')) {
    fail(401, 'Setup failed');
  }
  const credential = await verifyAttestation(
    input.attestation as Parameters<typeof verifyAttestation>[0],
    challenge.challenge
  ).catch(() => {
    fail(401, 'Setup failed');
  });
  // transaction 内で operator 数を再確認し、競合する初回作成を拒否する。
  const operator = await getAdminPrisma().$transaction(
    async (tx) => {
      if ((await operatorModel.countOperators(tx)) !== 0) fail(409, 'Bootstrap is not allowed');
      const created = await operatorModel.createInitialAdminOperator(tx, {
        email: challenge.email,
        displayName: challenge.displayName ?? challenge.email,
      });
      await passkeyModel.addOperatorPasskey(tx, {
        operatorId: created.id,
        credentialHandle: credential.credentialHandle,
        publicKey: credential.publicKey,
        signCount: BigInt(credential.signCount),
        aaguid: credential.aaguid,
        backupEligible: credential.backupEligible,
        backupState: credential.backupState,
        transports: credential.transports,
      });
      return created;
    },
    { isolationLevel: 'Serializable' }
  );
  return sessionRedirectResponse(operator, valkey);
};
