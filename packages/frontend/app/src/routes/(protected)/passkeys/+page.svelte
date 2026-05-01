<script lang="ts">
  import { usePasskeyManagement } from '@www-template/domain/auth/passkey/management';
  import PasskeyList from '../../../lib/profiles/PasskeyList.svelte';

  const { data, actions } = usePasskeyManagement();

  let otp = $state<string | null>(null);
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
    try {
      await actions.deletePasskey(id);
    } catch {
      localError = 'パスキーの削除に失敗しました。';
    }
  }

  async function handleIssueOtp(): Promise<void> {
    localError = null;
    otp = null;
    try {
      const result = await actions.issueOtp();
      if (result !== null) {
        otp = result;
      }
    } catch {
      localError = 'OTP の発行に失敗しました。';
    }
  }
</script>

<PasskeyList
  passkeys={data.passkeys}
  loading={data.loading}
  error={displayError}
  {otp}
  onAddPasskey={handleAddPasskey}
  onDeletePasskey={handleDeletePasskey}
  onIssueOtp={handleIssueOtp}
/>
