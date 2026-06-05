<script lang="ts">
  /**
   * ログインと端末ページ。
   * パスキー管理とデバイス/セッション管理を統合する。
   * settings 配下に配置し、トップレベルナビからは外す。
   */
  import { goto } from '$app/navigation';

  import { useAccount } from '@www-template/domain';
  import { usePasskeyManagement } from '@www-template/domain/auth/passkey/management';
  import { useDeviceManager, type DeviceManagerErrorCode } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { Button, Separator } from '@www-template/ui/components';

  import PasskeyList from '$lib/profiles/PasskeyList.svelte';
  import { DeviceManager } from '../../../../components/device-manager';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const { data: sessionData } = useAuthSession();
  const { data: deviceData, actions: deviceActions } = useDeviceManager();
  const { data: passkeyData, actions: passkeyActions } = usePasskeyManagement();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  /** パスキー操作のローカルエラー。 */
  let localPasskeyError = $state<string | null>(null);

  /** パスキー関連の表示用エラー。 */
  let displayPasskeyError = $derived(passkeyData.error ?? localPasskeyError);

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

  // パスキー一覧の初期読み込み
  if (typeof window !== 'undefined') {
    void initPasskeys();
  }

  async function initPasskeys(): Promise<void> {
    try {
      await passkeyActions.listPasskeys();
    } catch {
      localPasskeyError = 'passkeysListLoadFailed';
    }
  }

  async function handleAddPasskey(): Promise<void> {
    localPasskeyError = null;
    try {
      await passkeyActions.addPasskey();
    } catch {
      localPasskeyError = 'passkeyAddFailed';
    }
  }

  async function handleDeletePasskey(id: string): Promise<void> {
    localPasskeyError = null;
    const session = await passkeyActions.performReauth('passkey-delete');
    if (session === null) {
      localPasskeyError = 'reauthRequired';
      return;
    }
    try {
      await passkeyActions.deletePasskey(id, session);
    } catch {
      localPasskeyError = 'passkeyDeleteFailed';
    } finally {
      passkeyActions.clearReauthSession();
    }
  }

  async function handleSendDeviceLink(): Promise<void> {
    localPasskeyError = null;
    const session = await passkeyActions.performReauth('device-link');
    if (session === null) {
      localPasskeyError = 'reauthRequired';
      return;
    }
    try {
      const result = await passkeyActions.sendDeviceLink(session);
      if (result !== true) {
        localPasskeyError = 'deviceLinkSendFailed';
      }
    } catch {
      localPasskeyError = 'deviceLinkSendFailed';
    } finally {
      passkeyActions.clearReauthSession();
    }
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

<section class="flex flex-col gap-6 p-6">
  <header class="flex flex-col gap-2 border-b border-border pb-4">
    <h1 class="text-2xl font-bold">{i18n.t('settings.signInAndDevices')}</h1>
  </header>

  <div class="flex flex-col gap-6">
    <PasskeyList
      passkeys={passkeyData.passkeys}
      loading={passkeyData.loading}
      error={displayPasskeyError}
      deviceLinkSent={passkeyData.deviceLinkSent}
      onAddPasskey={handleAddPasskey}
      onDeletePasskey={handleDeletePasskey}
      onSendDeviceLink={handleSendDeviceLink}
    />

    <Separator />

    <DeviceManager
      devices={deviceData.state.devices}
      currentSessionId={sessionData.state.activeSessionId ?? ''}
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
        logoutButtonAriaLabel: (deviceName: string) =>
          i18n.t('device-manager.logoutButtonAriaLabel', { deviceName }),
        logoutButtonText: i18n.t('device-manager.logoutButtonText'),
        revokeOthersButtonText: i18n.t('device-manager.revokeOthersButtonText'),
      }}
    />
  </div>

  <div class="flex flex-col gap-2 border-t border-border pt-4">
    <Button variant="outline" href="/settings">
      {i18n.t('settings.title')}
    </Button>
  </div>
</section>
