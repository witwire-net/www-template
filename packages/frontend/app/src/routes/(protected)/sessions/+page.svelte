<script lang="ts">
  import { goto } from '$app/navigation';

  import { useAuthSession } from '@www-template/domain/auth/session';
  import { DeviceManager } from '@www-template/ui/components';

  import type { DeviceSession } from '@www-template/ui/components/device-manager';

  const { data, actions } = useAuthSession();

  let loading = $state(false);
  let localError = $state<string | null>(null);
  let devices = $state<DeviceSession[]>([]);

  /** ページマウント時にデバイス一覧を取得する。 */
  if (typeof window !== 'undefined') {
    void loadDevices();
  }

  async function loadDevices(): Promise<void> {
    loading = true;
    localError = null;
    try {
      const result = await actions.listDevices();
      if (result === null) {
        localError = 'デバイス一覧の取得に失敗しました。';
        devices = [];
      } else {
        devices = result;
      }
    } catch {
      localError = 'デバイス一覧の取得に失敗しました。';
      devices = [];
    } finally {
      loading = false;
    }
  }

  async function handleRevoke(sessionId: string): Promise<void> {
    localError = null;
    try {
      const ok = await actions.revokeDevice(sessionId);
      if (!ok) {
        localError = 'デバイスのログアウトに失敗しました。';
        return;
      }
      // ローカル一覧から該当デバイスを除去して即座に反映する
      devices = devices.filter((d) => d.sessionId !== sessionId);
      // 現在のセッションをログアウトした場合、state が anonymous に遷移するため login へ移動
      if (data.state.phase === 'anonymous') {
        await goto('/login');
      }
    } catch {
      localError = 'デバイスのログアウトに失敗しました。';
    }
  }

  async function handleRevokeOthers(): Promise<void> {
    localError = null;
    try {
      const ok = await actions.revokeOtherDevices();
      if (!ok) {
        localError = '他のデバイスのログアウトに失敗しました。';
        return;
      }
      // ローカル一覧から現在のデバイスのみを残す
      devices = devices.filter((d) => d.sessionId === data.state.activeSessionId);
    } catch {
      localError = '他のデバイスのログアウトに失敗しました。';
    }
  }
</script>

<DeviceManager
  {devices}
  currentSessionId={data.state.activeSessionId ?? ''}
  {loading}
  error={localError}
  onRevoke={handleRevoke}
  onRevokeOthers={handleRevokeOthers}
/>
