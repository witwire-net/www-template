<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { AuthPanel, Button } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { formatAuthError, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data, actions } = useRecoveryFlow();
  const initialLocale = resolveUnauthenticatedLocale();
  const i18n = useI18n(initialLocale);

  const errorMessage = $derived(formatAuthError(i18n, data.state.error, initialLocale));

  async function handleRegisterPasskey() {
    const result = await actions.registerRecoveryPasskey();
    if (result === null && data.state.phase !== 'registering' && data.state.error === null) {
      await goto('/');
    }
  }

  /*
   * recovery session は domain singleton state で共有し、sessionStorage には保存しない。
   * 直接アクセスやリロードで state が失われた場合は安全に復旧導線へ戻す。
   */
  if (typeof window !== 'undefined' && data.state.recoverySession === null) {
    void goto('/login/recovery');
  }
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <h1 class="auth-shell__heading">
        {data.state.kind === 'device-link'
          ? i18n.t('common.recoveryRegisterNewDeviceTitle')
          : i18n.t('common.recoveryRegisterReissueTitle')}
      </h1>
      <p class="auth-shell__body">
        {data.state.kind === 'device-link'
          ? i18n.t('common.recoveryRegisterNewDeviceDescription')
          : i18n.t('common.recoveryRegisterReissueDescription')}
      </p>
    </div>

    {#if errorMessage}
      <p class="auth-shell__error" role="alert">{errorMessage}</p>
    {/if}

    <div class="auth-shell__actions">
      <Button
        size="lg"
        type="button"
        disabled={data.state.phase === 'registering'}
        onclick={handleRegisterPasskey}
      >
        {#if data.state.phase === 'registering'}
          {i18n.t('common.recoveryRegisterSubmitting')}
        {:else}
          {i18n.t('common.recoveryRegisterSubmit')}
        {/if}
      </Button>

      <a class="auth-shell__link justify-center" href="/login/recovery">
        {i18n.t('common.recoveryRegisterRetry')}
      </a>
    </div>
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('common.recoveryRegisterBackToPublic')}</a>
  {/snippet}
</AuthLayout>
