import { useAuthSession } from '../session/hook.svelte';

import type { AuthRouteIntent } from '../types';
import type { BootstrapPhase } from '../session/hook.svelte';

interface SessionGuardData {
  state: ReturnType<typeof useAuthSession>['data']['state'];
  /** context index bootstrap の進行状態。 */
  bootstrapPhase: { value: BootstrapPhase };
}

interface SessionGuardActions {
  redirectIfRequired: () => AuthRouteIntent | null;
}

interface SessionGuardOptions {
  readPathname: () => string;
  redirectTo: (intent: AuthRouteIntent) => void;
  sessionExpiredPath?: Extract<AuthRouteIntent, '/session-expired'>;
  accountSuspendedPath?: Extract<AuthRouteIntent, '/account-suspended'>;
}

const SESSION_EXPIRED_PATH: Extract<AuthRouteIntent, '/session-expired'> = '/session-expired';
const ACCOUNT_SUSPENDED_PATH: Extract<AuthRouteIntent, '/account-suspended'> = '/account-suspended';

const resolveSessionGuardIntent = (
  pathname: string,
  state: SessionGuardData['state'],
  bootstrapPhase: { value: BootstrapPhase },
  authSessionActions: ReturnType<typeof useAuthSession>['actions'],
  sessionExpiredPath: Extract<AuthRouteIntent, '/session-expired'>,
  accountSuspendedPath: Extract<AuthRouteIntent, '/account-suspended'>
): AuthRouteIntent | null => {
  // session-expired / account-suspended ページ上では redirect を抑止してループを防ぐ
  if (pathname === sessionExpiredPath || pathname === accountSuspendedPath) {
    return null;
  }

  // context index bootstrap が進行中の場合、復元完了前に `/login` へ飛ばない。
  // bootstrap 完了後に guard が再評価されるため、ここで待機する。
  if (bootstrapPhase.value === 'pending') {
    return null;
  }

  if (state.phase === 'session-expired') {
    return sessionExpiredPath;
  }

  if (state.phase === 'account-suspended') {
    return accountSuspendedPath;
  }

  // bootstrap 完了後に anonymous または session なし → missing session として `/login` へ
  if (state.phase === 'anonymous' || state.session === null) {
    return authSessionActions.handleMissingSession();
  }

  return null;
};

/**
 * auth session phase を監視し、必要な route へ fail-close redirect する。
 *
 * - redirect 実行は app 層 callback に委譲し、domain は intent 判定に集中する。
 * - `/session-expired` 上では redirect を抑止してループを防ぐ。
 */
function useSessionGuard(options: SessionGuardOptions): {
  data: SessionGuardData;
  actions: SessionGuardActions;
} {
  const { data, actions: authSessionActions } = useAuthSession();
  const sessionExpiredPath = options.sessionExpiredPath ?? SESSION_EXPIRED_PATH;
  const accountSuspendedPath = options.accountSuspendedPath ?? ACCOUNT_SUSPENDED_PATH;

  const actions: SessionGuardActions = {
    redirectIfRequired: () => {
      const intent = resolveSessionGuardIntent(
        options.readPathname(),
        data.state,
        data.bootstrapPhase,
        authSessionActions,
        sessionExpiredPath,
        accountSuspendedPath
      );

      if (intent !== null) {
        options.redirectTo(intent);
      }

      return intent;
    },
  };

  $effect(() => {
    actions.redirectIfRequired();
  });

  return {
    data: {
      state: data.state,
      bootstrapPhase: data.bootstrapPhase,
    },
    actions,
  };
}

export type { SessionGuardActions, SessionGuardData, SessionGuardOptions };
export { useSessionGuard };
