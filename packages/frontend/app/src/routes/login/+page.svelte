<script lang="ts">
  import { goto } from '$app/navigation';

  import { usePasskeyLogin } from '@www-template/domain/auth/passkey';
  import { AuthPanel, Button } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import {
    formatAuthError,
    persistAppLocale,
    resolveUnauthenticatedLocale,
    useI18n,
  } from '$lib/i18n';

  const { data, actions } = usePasskeyLogin();

  // 認証前 fallback locale で i18n を初期化する
  const initialLocale = resolveUnauthenticatedLocale();
  persistAppLocale(initialLocale);
  const i18n = useI18n(initialLocale);

  const errorMessage = $derived(formatAuthError(i18n, data.state.error, initialLocale));

  async function handlePasskeySignIn() {
    const result = await actions.signInWithPasskey();
    if (result !== null) {
      await goto(result);
      return;
    }
    if (result === null && data.state.lastSession !== null) {
      await goto('/');
    }
  }
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <h1 class="auth-shell__heading">{i18n.t('login.login')}</h1>
      <p class="auth-shell__body">{i18n.t('login.passkeyPrompt')}</p>
    </div>

    {#if errorMessage}
      <p class="auth-shell__error" role="alert">{errorMessage}</p>
    {/if}

    <div class="auth-shell__actions">
      <Button
        type="button"
        size="lg"
        disabled={data.state.isSubmitting}
        onclick={handlePasskeySignIn}
      >
        {#if data.state.isSubmitting}
          {i18n.t('login.signingIn')}
        {:else}
          {i18n.t('login.signInButton')}
        {/if}
      </Button>

      <a class="auth-shell__link justify-center" href="/login/recovery">
        {i18n.t('login.lostPasskey')}
      </a>
    </div>
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('login.backToPublic')}</a>
  {/snippet}
</AuthLayout>
