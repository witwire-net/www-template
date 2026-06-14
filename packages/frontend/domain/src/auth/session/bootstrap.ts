import {
  clearContextIndex,
  createEmptyContextIndex,
  readContextIndex,
  toContextIndexEntry,
  upsertContextEntry,
  writeContextIndex,
} from './context_index';
import { decodeAccessToken } from './token_state';

import type { AuthSessionState, AuthSessionSummary } from '../types';

/** context index bootstrap の進行状態を表す。 */
export type BootstrapPhase = 'pending' | 'done';

interface BootstrapRefreshSuccessData {
  requestId: string;
  accessToken: string;
  account: {
    accountId: string;
    passkeyCredentialId: string;
  };
  sessionId: string;
  expiresAt: string;
}

interface BootstrapRefreshSuccessResponse {
  status: 200;
  data: BootstrapRefreshSuccessData;
}

interface BootstrapRefreshFallbackResponse {
  status: number;
  data: unknown;
}

type BootstrapRefreshResponse = BootstrapRefreshSuccessResponse | BootstrapRefreshFallbackResponse;

type BootstrapRefreshContext = (authContextId: string) => Promise<BootstrapRefreshResponse>;

interface BootstrapSessionsFromContextIndexOptions {
  authState: AuthSessionState;
  bootstrapPhase: { value: BootstrapPhase };
  refreshContext: BootstrapRefreshContext;
}

function hasBootstrapRefreshSuccessData(
  response: BootstrapRefreshResponse
): response is BootstrapRefreshSuccessResponse {
  return (
    response.status === 200 &&
    typeof response.data === 'object' &&
    response.data !== null &&
    'accessToken' in response.data
  );
}

/**
 * bootstrapSessionsFromContextIndex は Product origin-local の context index からセッションを復元する。
 *
 * 引数:
 *   - options.authState: 復元した in-memory bearer session を反映する認証状態。
 *   - options.bootstrapPhase: guard が bootstrap 完了を判定するための共有 phase オブジェクト。
 *   - options.refreshContext: authContextId ごとに HttpOnly Cookie refresh を実行し、access token を再取得する関数。
 *
 * 戻り値:
 *   - Promise<void>: 復元できた場合は authState と context index を更新し、復元できない場合は index を削除して完了する。
 *
 * エラー:
 *   - 個別 context の refresh 失敗は該当 entry を採用しないだけで外へ投げない。
 *   - localStorage の読み書き失敗は context_index 側で fail-close され、この関数からは投げない。
 *
 * 使用例:
 *
 * ```ts
 * void bootstrapSessionsFromContextIndex({
 *   authState: state,
 *   bootstrapPhase,
 *   refreshContext: (authContextId) => refreshToken(authContextId, undefined, requestInit),
 * });
 * ```
 */
async function bootstrapSessionsFromContextIndex({
  authState,
  bootstrapPhase,
  refreshContext,
}: BootstrapSessionsFromContextIndexOptions): Promise<void> {
  try {
    // Step 1: context index は secret を含まない hint なので、存在しない場合は復元せずに完了する。
    const index = readContextIndex();
    if (index == null || index.entries.length === 0) {
      return;
    }

    const restoredSessions: AuthSessionSummary[] = [];
    let restoredActiveSession: AuthSessionSummary | null = null;

    // Step 2: 各 entry を server refresh で再検証し、成功した context だけを in-memory session 候補にする。
    for (const entry of index.entries) {
      try {
        const response = await refreshContext(entry.authContextId);
        if (hasBootstrapRefreshSuccessData(response)) {
          const { accessToken, account, sessionId, expiresAt } = response.data;
          const accountId = account.accountId;
          const claims = decodeAccessToken(accessToken);
          if (claims?.accountId !== accountId || claims.sessionId !== sessionId) {
            continue;
          }
          const restoredSession: AuthSessionSummary = {
            requestId: response.data.requestId,
            authContextId: entry.authContextId,
            accountId,
            passkeyCredentialId: account.passkeyCredentialId,
            sessionId,
            accessToken,
            expiresAt,
          };
          restoredSessions.push(restoredSession);
          if (index.activeAuthContextId === entry.authContextId) {
            restoredActiveSession = restoredSession;
          }
        }
      } catch {
        // Step 3: refresh に失敗した entry は信頼できないため、復元対象から除外する。
      }
    }

    if (restoredSessions.length > 0) {
      // Step 4: 同一 accountId の重複は後方 entry を優先して 1 件へ正規化する。
      const dedupedSessions: AuthSessionSummary[] = [];
      const seenAccountIds = new Set<string>();
      for (const session of [...restoredSessions].reverse()) {
        if (!seenAccountIds.has(session.accountId)) {
          seenAccountIds.add(session.accountId);
          dedupedSessions.unshift(session);
        }
      }

      authState.sessions = dedupedSessions;
      // Step 5: 直前の active session が残っていれば維持し、無ければ先頭 session を active にする。
      const active =
        restoredActiveSession != null &&
        dedupedSessions.some((session) => session.sessionId === restoredActiveSession.sessionId)
          ? restoredActiveSession
          : dedupedSessions[0];
      authState.session = active;
      authState.activeSessionId = active.sessionId;
      authState.phase = 'authenticated';
      authState.routeIntent = '/login';
      authState.lastFailure = null;
      authState.lastError = null;

      // Step 6: refresh に成功した session だけで context index を再構築し、失敗 entry を残さない。
      const nextIndex = createEmptyContextIndex();
      for (const session of dedupedSessions) {
        upsertContextEntry(
          nextIndex,
          toContextIndexEntry(session, session.expiresAt),
          session.sessionId === active.sessionId
        );
      }
      writeContextIndex(nextIndex);
    } else {
      // Step 7: どの entry も復元できない場合は、次回起動で同じ失敗を繰り返さないよう index を破棄する。
      clearContextIndex();
    }
  } finally {
    // Step 8: 成否にかかわらず bootstrap 完了を記録し、guard が redirect 判断を再評価できるようにする。
    bootstrapPhase.value = 'done';
  }
}

export { bootstrapSessionsFromContextIndex };
