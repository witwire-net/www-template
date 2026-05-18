import { getProductPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { searchAccounts } from '$lib/server/services/accounts/search';

import type { ServerLoad } from '@sveltejs/kit';

const PAGE_SIZE = 20;

function parsePage(value: string | null): number {
  // URL クエリから 1-based page を安全に復元し、不正値は先頭ページへ戻す。
  const parsed = Number(value ?? '1');
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

/**
 * アカウント一覧ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @param url 現在のリクエスト URL
 * @returns 一覧表示に必要な検索結果
 */
export const load: ServerLoad = async ({ locals, url }) => {
  // 顧客一覧は機微情報を含むため、DB 現在ロールで読み取り権限を検証する。
  requirePermission(locals.operator, 'accounts:read');
  const query = url.searchParams.get('query') ?? '';
  const status = url.searchParams.get('status') ?? '';
  const page = parsePage(url.searchParams.get('page'));
  const result = await searchAccounts(await getProductPrisma(), {
    query: query !== '' ? query : undefined,
    status: status !== '' ? status : undefined,
    page,
    limit: PAGE_SIZE,
  });

  return { ...result, filters: { query, status } };
};
