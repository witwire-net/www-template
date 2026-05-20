<script lang="ts">
  import { goto } from '$app/navigation';

  import { useAccount } from '@www-template/domain';
  import { useDeviceManager, type DeviceManagerErrorCode } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { DeviceManager } from '../../../components/device-manager';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data } = useAuthSession();
  const { data: accountData } = useAccount();
  const { data: deviceData, actions: deviceActions } = useDeviceManager();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

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

  async function handleRevoke(sessionId: string): Promise<void> {
    const ok = await deviceActions.revokeDevice(sessionId);
    if (ok && data.state.phase === 'anonymous') {
      await goto('/login');
    }
  }

  async function handleRevokeOthers(): Promise<void> {
    const ok = await deviceActions.revokeOtherDevices();
    if (ok && data.state.phase === 'anonymous') {
      await goto('/login');
    }
  }

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
</script>

  <DeviceManager
  devices={deviceData.state.devices}
  currentSessionId={data.state.activeSessionId ?? ''}
  loading={deviceData.state.loading}
  error={deviceError}
  onRevoke={handleRevoke}
  onRevokeOthers={handleRevokeOthers}
  {formatDateTime}
  labels={{
    sectionAriaLabel: i18n.t('device-manager.sectionAriaLabel'),
    loadingText: i18n.t('device-manager.loadingText'),
    emptyText: i18n.t('device-manager.emptyText'),
    loginAtLabel: i18n.t('device-manager.loginAtLabel'),
    lastActiveAtLabel: i18n.t('device-manager.lastActiveAtLabel'),
    currentDeviceBadge: i18n.t('device-manager.currentDeviceBadge'),
    logoutButtonAriaLabel: (deviceName) =>
      i18n.t('device-manager.logoutButtonAriaLabel', { deviceName }),
    logoutButtonText: i18n.t('device-manager.logoutButtonText'),
    revokeOthersButtonText: i18n.t('device-manager.revokeOthersButtonText'),
  }}
/>
