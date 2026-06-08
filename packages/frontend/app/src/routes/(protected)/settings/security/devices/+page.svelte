<script lang="ts">
  /**
   * 端末/セッション管理ページ。
   * 1ページ1概念: ログイン中の端末一覧と個別ログアウトを担当する。
   * 他端末一括ログアウトは Danger zone として通常操作から分離する。
   */
  import { goto } from '$app/navigation';

  import { useAccount } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { useDeviceManager, type DeviceManagerErrorCode } from '@www-template/domain';
  import { Alert, AlertDescription, AlertTitle } from '@www-template/ui/components/alert';
  import { Button, Separator } from '@www-template/ui/components';

  import { DeviceManager } from '../../../../../components/device-manager';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const { data: sessionData } = useAuthSession();
  const { data: deviceData, actions: deviceActions } = useDeviceManager();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  /** デバイス関連のエラーメッセージ。 */
  const deviceError = $derived.by(() => {
    const errorCode = deviceData.state.errorCode;
    if (errorCode === null) {
      return null;
    }

    const messages: Record<DeviceManagerErrorCode, string> = {
      load: i18n.t('device-manager.error'),
      revoke: i18n.t('device-manager.logoutError'),
      'revoke-others': i18n.t('device-manager.revokeOthersError'),
    };

    return messages[errorCode];
  });

  /** 現在のデバイスを除く他のデバイスが存在するかどうか。 */
  let hasOtherDevices = $derived(
    deviceData.state.devices.some((d) => d.sessionId !== (sessionData.state.activeSessionId ?? ''))
  );

  function formatDateTime(iso: string): string {
    const d = new Date(iso);
    return i18n.formatters.dateTime(d, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  async function handleRevoke(sessionId: string): Promise<void> {
    const ok = await deviceActions.revokeDevice(sessionId);
    if (ok && sessionData.state.phase === 'anonymous') {
      await goto('/login');
    }
  }

  async function handleRevokeOthers(): Promise<void> {
    const ok = await deviceActions.revokeOtherDevices();
    if (ok && sessionData.state.phase === 'anonymous') {
      await goto('/login');
    }
  }
</script>

<section class="flex flex-col gap-6">
  <!-- 通常操作: 端末一覧と個別ログアウト -->
  <DeviceManager
    devices={deviceData.state.devices}
    currentSessionId={sessionData.state.activeSessionId ?? ''}
    loading={deviceData.state.loading}
    error={deviceError}
    onRevoke={handleRevoke}
    onRevokeOthers={handleRevokeOthers}
    showRevokeOthers={false}
    {formatDateTime}
    labels={{
      sectionAriaLabel: i18n.t('device-manager.sectionAriaLabel'),
      loadingText: i18n.t('device-manager.loadingText'),
      emptyText: i18n.t('device-manager.emptyText'),
      loginAtLabel: i18n.t('device-manager.loginAtLabel'),
      lastActiveAtLabel: i18n.t('device-manager.lastActiveAtLabel'),
      currentDeviceBadge: i18n.t('device-manager.currentDeviceBadge'),
      logoutButtonAriaLabel: (deviceName: string) =>
        i18n.t('device-manager.logoutButtonAriaLabel', { deviceName }),
      logoutButtonText: i18n.t('device-manager.logoutButtonText'),
      revokeOthersButtonText: i18n.t('device-manager.revokeOthersButtonText'),
    }}
  />

  <Separator />

  <!-- Danger zone: 他端末一括ログアウト（通常操作から分離） -->
  <Alert variant="destructive">
    <AlertTitle>{i18n.t('settings.dangerZone')}</AlertTitle>
    <AlertDescription>
      {i18n.t('settings.dangerZoneDescription')}
    </AlertDescription>
    <div class="mt-3">
      <Button
        variant="destructive"
        size="sm"
        disabled={deviceData.state.loading || !hasOtherDevices}
        onclick={handleRevokeOthers}
      >
        {i18n.t('device-manager.revokeOthersButtonText')}
      </Button>
    </div>
  </Alert>
</section>
