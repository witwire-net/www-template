import { json } from '@sveltejs/kit';

import { consumeChallenge } from '$lib/server/infrastructure/auth/operator.js';
import { verifyAttestation } from '$lib/server/infrastructure/auth/registration.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import {
  NO_STORE_HEADERS,
  fail,
  registrationFinishRequestSchema,
  parseJson,
  requireAuthenticatedOperator,
  requireValkey,
  serializePasskey,
} from '$lib/server/services/auth/routes.js';

import type { RequestHandler } from '@sveltejs/kit';

/**
 * 認証済みオペレーターの passkey 追加完了 API。
 *
 * @param event SvelteKit request event
 * @returns 登録された passkey metadata
 */
export const POST: RequestHandler = async (event) => {
  // 認証済み本人と challenge に保存した operatorId を必ず照合する。
  const localOperator = requireAuthenticatedOperator(event);
  const input = await parseJson(event, registrationFinishRequestSchema);
  const valkey = await requireValkey();
  const challenge = await consumeChallenge(input.challengeId, 'passkey-add', valkey).catch(() => {
    fail(401, 'Registration failed');
  });
  if (challenge.operatorId !== localOperator.id) fail(403, 'Forbidden');
  // attestation 検証済み credential だけを passkey table に追加する。
  const credential = await verifyAttestation(
    input.attestation as Parameters<typeof verifyAttestation>[0],
    challenge.challenge
  ).catch(() => {
    fail(401, 'Registration failed');
  });
  const passkey = await passkeyModel.addOperatorPasskey(getAdminPrisma(), {
    operatorId: localOperator.id,
    credentialHandle: credential.credentialHandle,
    publicKey: credential.publicKey,
    signCount: BigInt(credential.signCount),
    aaguid: credential.aaguid,
    backupEligible: credential.backupEligible,
    backupState: credential.backupState,
    transports: credential.transports,
  });
  return json({ passkey: serializePasskey(passkey) }, { status: 201, headers: NO_STORE_HEADERS });
};
