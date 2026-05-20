import { onMount } from 'svelte';

import { useAuthSession } from './auth/session';

/**
 * DeviceManager が扱うデバイスセッションです。
 */
export interface DeviceSession {
  readonly sessionId: string;
  readonly deviceName: string;
  readonly loginAt: string;
  readonly lastActiveAt: string;
  readonly ipHash: string;
  readonly isCurrentSession: boolean;
}

/**
 * DeviceManager が扱うエラー種別です。
 */
export type DeviceManagerErrorCode = 'load' | 'revoke' | 'revoke-others';

/**
 * DeviceManager の view state です。
 */
export interface DeviceManagerData {
  readonly state: {
    readonly devices: DeviceSession[];
    readonly loading: boolean;
    readonly errorCode: DeviceManagerErrorCode | null;
  };
}

/**
 * DeviceManager の操作です。
 */
export interface DeviceManagerActions {
  /** 一覧を再取得します。 */
  refresh: () => Promise<void>;
  /** 指定セッションを取り消します。 */
  revokeDevice: (sessionId: string) => Promise<boolean>;
  /** 他のセッションを取り消します。 */
  revokeOtherDevices: () => Promise<boolean>;
}

/**
 * 認証済みアプリのデバイス管理 state を扱います。
 */
export function useDeviceManager(): { data: DeviceManagerData; actions: DeviceManagerActions } {
  const { data: sessionData, actions: sessionActions } = useAuthSession();

  const state = $state({
    devices: [] as DeviceSession[],
    loading: false,
    errorCode: null as DeviceManagerErrorCode | null,
  });

  const loadDevices = async (): Promise<void> => {
    state.loading = true;
    state.errorCode = null;
    try {
      const result = await sessionActions.listDevices();
      state.devices = result ?? [];
      if (result === null) {
        state.errorCode = 'load';
      }
    } catch {
      state.devices = [];
      state.errorCode = 'load';
    } finally {
      state.loading = false;
    }
  };

  onMount(() => {
    void loadDevices();
  });

  const revokeDevice = async (sessionId: string): Promise<boolean> => {
    state.errorCode = null;
    try {
      const ok = await sessionActions.revokeDevice(sessionId);
      if (!ok) {
        state.errorCode = 'revoke';
        return false;
      }

      state.devices = state.devices.filter((device) => device.sessionId !== sessionId);
      return true;
    } catch {
      state.errorCode = 'revoke';
      return false;
    }
  };

  const revokeOtherDevices = async (): Promise<boolean> => {
    state.errorCode = null;
    try {
      const ok = await sessionActions.revokeOtherDevices();
      if (!ok) {
        state.errorCode = 'revoke-others';
        return false;
      }

      state.devices = state.devices.filter(
        (device) => device.sessionId === sessionData.state.activeSessionId
      );
      return true;
    } catch {
      state.errorCode = 'revoke-others';
      return false;
    }
  };

  return {
    data: { state },
    actions: {
      refresh: loadDevices,
      revokeDevice,
      revokeOtherDevices,
    },
  };
}
