import { statusApi } from '@www-template-frontend/api';

import { createStatusInitialState, toStatusErrorMessage } from '../../status/statusState';

import type { StatusState } from 'types';

interface StatusData {
  state: StatusState;
}

interface StatusActions {
  refresh: () => Promise<void>;
}

/**
 * 公開向け status 状態を扱う stateful な Svelte domain composable。
 */
function useStatus(): { data: StatusData; actions: StatusActions } {
  const state = $state<StatusState>(createStatusInitialState());

  const actions: StatusActions = {
    refresh: async () => {
      state.error = undefined;
      state.isLoading = true;

      try {
        const status = await statusApi.get();

        state.error = undefined;
        state.isLoading = false;
        state.message = status.message;
        state.timestamp = status.timestamp;
      } catch (error: unknown) {
        state.error = toStatusErrorMessage(error);
        state.isLoading = false;
      }
    },
  };

  if (!import.meta.env.SSR) {
    void actions.refresh();
  }

  return {
    data: {
      state,
    },
    actions,
  };
}

export type { StatusActions, StatusData };
export { useStatus };
