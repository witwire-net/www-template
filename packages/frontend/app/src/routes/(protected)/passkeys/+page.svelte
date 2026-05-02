<script lang="ts">
  import { usePasskeyManagement } from '@www-template/domain/auth/passkey/management';
  import PasskeyList from '../../../lib/profiles/PasskeyList.svelte';

  const { data, actions } = usePasskeyManagement();

  /*
   * OTP 発行状態を boolean で保持する。
   * 平文 OTP 値は画面に表示しないため、値本体は保存せず発行有無のみを管理する。
   */
  let otpIssued = $state(false);
  let localError = $state<string | null>(null);

  let displayError = $derived(data.error ?? localError);

  if (typeof window !== 'undefined') {
    void initPasskeys();
  }

  async function initPasskeys(): Promise<void> {
    try {
      await actions.listPasskeys();
    } catch {
      localError = 'パスキー一覧の取得に失敗しました。';
    }
  }

  async function handleAddPasskey(): Promise<void> {
    localError = null;
    try {
      await actions.addPasskey();
    } catch {
      localError = 'パスキーの登録に失敗しました。';
    }
  }

  async function handleDeletePasskey(id: string): Promise<void> {
    localError = null;
    const session = await actions.performReauth('passkey-delete');
    if (session === null) {
      localError = '再認証が必要です。';
      return;
    }
    try {
      await actions.deletePasskey(id, session);
    } catch {
      localError = 'パスキーの削除に失敗しました。';
    } finally {
      actions.clearReauthSession();
    }
  }

  async function handleIssueOtp(): Promise<void> {
    localError = null;
    otpIssued = false;
    const session = await actions.performReauth('otp-issue');
    if (session === null) {
      localError = '再認証が必要です。';
      return;
    }
    try {
      const result = await actions.issueOtp(session);
      /*
       * issueOtp が成功して true を返した場合、平文 OTP は UI に表示せず
       * 「発行済み」フラグのみを立ててメール送信済み案内を表示する。
       */
      if (result === true) {
        otpIssued = true;
      }
    } catch {
      localError = 'OTP の発行に失敗しました。';
    } finally {
      actions.clearReauthSession();
    }
  }
</script>

<PasskeyList
  passkeys={data.passkeys}
  loading={data.loading}
  error={displayError}
  {otpIssued}
  onAddPasskey={handleAddPasskey}
  onDeletePasskey={handleDeletePasskey}
  onIssueOtp={handleIssueOtp}
/>
