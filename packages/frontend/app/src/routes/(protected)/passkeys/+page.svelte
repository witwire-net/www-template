<script lang="ts">
  import { usePasskeyManagement } from '@www-template/domain/auth/passkey/management';
  import PasskeyList from '../../../lib/profiles/PasskeyList.svelte';

  /**
   * パスキー管理ページ。
   * 登録済みパスキーの一覧表示、追加、削除、および新しい端末へのデバイスリンク送信を行う。
   * ユーザー向けエラーは PasskeyList 側で現在の locale に合わせて翻訳する。
   */
  const { data, actions } = usePasskeyManagement();

  let localError = $state<string | null>(null);

  let displayError = $derived(data.error ?? localError);

  if (typeof window !== 'undefined') {
    void initPasskeys();
  }

  async function initPasskeys(): Promise<void> {
    try {
      await actions.listPasskeys();
    } catch {
      localError = 'passkeysListLoadFailed';
    }
  }

  async function handleAddPasskey(): Promise<void> {
    localError = null;
    try {
      await actions.addPasskey();
    } catch {
      localError = 'passkeyAddFailed';
    }
  }

  async function handleDeletePasskey(id: string): Promise<void> {
    localError = null;
    const session = await actions.performReauth('passkey-delete');
    if (session === null) {
      localError = 'reauthRequired';
      return;
    }
    try {
      await actions.deletePasskey(id, session);
    } catch {
      localError = 'passkeyDeleteFailed';
    } finally {
      actions.clearReauthSession();
    }
  }

  async function handleSendDeviceLink(): Promise<void> {
    localError = null;
    const session = await actions.performReauth('device-link');
    if (session === null) {
      localError = 'reauthRequired';
      return;
    }
    try {
      const result = await actions.sendDeviceLink(session);
      /*
       * sendDeviceLink が成功して true を返した場合、
       * デバイスリンク送信済みフラグを立ててメール送信済み案内を表示する。
       */
      if (result !== true) {
        localError = 'deviceLinkSendFailed';
      }
    } catch {
      localError = 'deviceLinkSendFailed';
    } finally {
      actions.clearReauthSession();
    }
  }
</script>

<PasskeyList
  passkeys={data.passkeys}
  loading={data.loading}
  error={displayError}
  deviceLinkSent={data.deviceLinkSent}
  onAddPasskey={handleAddPasskey}
  onDeletePasskey={handleDeletePasskey}
  onSendDeviceLink={handleSendDeviceLink}
/>
