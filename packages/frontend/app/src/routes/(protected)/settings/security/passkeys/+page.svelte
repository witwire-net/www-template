<script lang="ts">
  /**
   * パスキー管理ページ。
   * 1ページ1概念: パスキーの表示・追加・削除のみを担当する。
   * 既存 PasskeyList コンポーネントを再利用する。
   */
  import { usePasskeyManagement } from '@www-template/domain/auth/passkey/management';

  import PasskeyList from '$lib/profiles/PasskeyList.svelte';

  const { data: passkeyData, actions: passkeyActions } = usePasskeyManagement();

  /** パスキー操作のローカルエラー。 */
  let localPasskeyError = $state<string | null>(null);

  /** パスキー関連の表示用エラー。 */
  let displayPasskeyError = $derived(passkeyData.error ?? localPasskeyError);

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
</script>

<section class="flex flex-col gap-4">
  <PasskeyList
    passkeys={passkeyData.passkeys}
    loading={passkeyData.loading}
    error={displayPasskeyError}
    deviceLinkSent={passkeyData.deviceLinkSent}
    onAddPasskey={handleAddPasskey}
    onDeletePasskey={handleDeletePasskey}
    onSendDeviceLink={handleSendDeviceLink}
  />
</section>
