import { verifyProtectedAdminRoute } from '../auth';

import type { AdminSessionState } from '../auth';

type AdminSessionRouteState = 'public' | 'checking' | 'authenticated' | 'blocked';

interface AdminSessionViewState {
  routeState: AdminSessionRouteState;
  session: AdminSessionState | null;
  verifiedPath: string | null;
}

interface AdminSessionData {
  state: AdminSessionViewState;
}

interface AdminSessionActions {
  verifyCurrentRoute: (verifiedPath: string) => Promise<void>;
  markPublicRoute: (path: string) => void;
}

interface AdminSessionOptions {
  readPath: () => string;
  isPublicPath: (path: string) => boolean;
  redirectToLogin: () => void;
}

function createInitialSessionViewState(): AdminSessionViewState {
  // protected route は検証完了まで child content を出さないため、初期状態を checking に固定する。
  return { routeState: 'checking', session: null, verifiedPath: null };
}

/**
 * Admin Console の protected route session を検証する domain composable です。
 *
 * route component は現在 path の読み取りと login 遷移 callback だけを渡し、
 * session refresh / current operator 検証 / 表示可否 state は domain に集約します。
 */
function useAdminSession(options: AdminSessionOptions): {
  data: AdminSessionData;
  actions: AdminSessionActions;
} {
  const state = $state<AdminSessionViewState>(createInitialSessionViewState());

  const actions: AdminSessionActions = {
    markPublicRoute: (path) => {
      // login/setup 系は operator session を要求せず、古い operator 表示も残さない。
      state.routeState = 'public';
      state.session = null;
      state.verifiedPath = path;
    },
    verifyCurrentRoute: async (verifiedPath) => {
      // 検証開始時点の path を記録し、遷移競合で古い結果を反映しないようにする。
      state.routeState = 'checking';
      state.verifiedPath = verifiedPath;

      const result = await verifyProtectedAdminRoute();
      if (state.verifiedPath !== verifiedPath || options.readPath() !== verifiedPath) return;

      if (result.status !== 'authenticated') {
        // 401/403 の詳細は domain result で丸め、UI には login 誘導だけを返す。
        state.session = null;
        state.routeState = 'blocked';
        options.redirectToLogin();
        return;
      }

      // Go Admin API が検証済みの operator/session だけを protected shell 表示へ渡す。
      state.session = result.session;
      state.routeState = 'authenticated';
    },
  };

  $effect(() => {
    // SvelteKit page state の path 変化を domain composable が監視し、route component から $effect を排除する。
    const currentPath = options.readPath();
    if (options.isPublicPath(currentPath)) {
      actions.markPublicRoute(currentPath);
      return;
    }

    // protected route は常に backend current operator 検証を通してから表示する。
    void actions.verifyCurrentRoute(currentPath);
  });

  return { data: { state }, actions };
}

export { useAdminSession };
