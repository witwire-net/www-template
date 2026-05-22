import Redis from 'ioredis';

import { getAdminAuthConfig } from '../config/env.js';

let adminValkey: Redis | null = null;

/**
 * 共有 Valkey infrastructure の Admin 用 logical DB に接続するクライアントを取得する。
 *
 * @returns Admin auth / rate-limit 用 logical DB へ向いた Valkey 接続
 */
export function getAdminValkey(): Redis {
  // 遅延初期化により、テストやビルド時に不要な外部接続を作らない。
  if (adminValkey === null) {
    const { adminValkeyUrl } = getAdminAuthConfig();
    adminValkey = new Redis(adminValkeyUrl, {
      maxRetriesPerRequest: 1,
      lazyConnect: true,
    });
  }
  return adminValkey;
}

/**
 * テストやプロセス終了時に Admin 用 Valkey 接続を閉じる。
 */
export async function disconnectAdminValkey(): Promise<void> {
  if (adminValkey !== null) {
    // quit 失敗時も参照は破棄し、次回呼び出しで新しい接続を作れるようにする。
    const client = adminValkey;
    adminValkey = null;
    await client.quit();
  }
}
