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

const SETUP_FAILED_MESSAGE = 'Setup failed';

/**
 * 追加オペレーターの passkey セットアップ完了 API。
 *
 * @param event SvelteKit request event
 * @returns session cookie 付き 303 redirect
 */
export const POST: RequestHandler = async (event) => {
  // start で検証済み token から作られた challenge だけを受け付ける。
  const input = await parseJson(event, registrationFinishRequestSchema);
  const valkey = await requireValkey();
  const challenge = await consumeChallenge(input.challengeId, 'operator-setup', valkey).catch(
    () => {
      fail(401, SETUP_FAILED_MESSAGE);
    }
  );
  const operator = await operatorModel.findOperatorById(getAdminPrisma(), challenge.operatorId);
  if (operator === null || !operator.isActive || operator.email !== challenge.email)
    fail(401, SETUP_FAILED_MESSAGE);
  const credential = await verifyAttestation(
    input.attestation as Parameters<typeof verifyAttestation>[0],
    challenge.challenge
  ).catch(() => {
    fail(401, SETUP_FAILED_MESSAGE);
  });
  // passkey 登録と setup token の条件付き消費を同一 transaction にし、token reuse を防ぐ。
  await getAdminPrisma().$transaction(
    async (tx) => {
      const existingPasskeys = await passkeyModel.listOperatorPasskeys(tx, operator.id);
      if (existingPasskeys.length > 0) fail(401, SETUP_FAILED_MESSAGE);
      const consumed = await operatorModel.consumeOperatorSetupToken(tx, operator.id, new Date());
      if (!consumed) fail(401, SETUP_FAILED_MESSAGE);
      await passkeyModel.addOperatorPasskey(tx, {
        operatorId: operator.id,
        credentialHandle: credential.credentialHandle,
        publicKey: credential.publicKey,
        signCount: BigInt(credential.signCount),
        aaguid: credential.aaguid,
        backupEligible: credential.backupEligible,
        backupState: credential.backupState,
        transports: credential.transports,
      });
    },
    { isolationLevel: 'Serializable' }
  );
  return sessionRedirectResponse(operator, valkey);
};
