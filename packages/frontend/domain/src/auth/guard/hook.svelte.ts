import { useAuthSession } from '../session/hook.svelte';

import type { AuthRouteIntent } from '../types';

interface SessionGuardData {
  state: ReturnType<typeof useAuthSession>['data']['state'];
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
  authSessionActions: ReturnType<typeof useAuthSession>['actions'],
  sessionExpiredPath: Extract<AuthRouteIntent, '/session-expired'>,
  accountSuspendedPath: Extract<AuthRouteIntent, '/account-suspended'>
): AuthRouteIntent | null => {
  if (pathname === sessionExpiredPath || pathname === accountSuspendedPath) {
    return null;
  }

  if (state.phase === 'session-expired') {
    return sessionExpiredPath;
  }

  if (state.phase === 'account-suspended') {
    return accountSuspendedPath;
  }

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
    },
    actions,
  };
}

export type { SessionGuardActions, SessionGuardData, SessionGuardOptions };
export { useSessionGuard };
