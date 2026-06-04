<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { AuthPanel, Button, Input, Label } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { formatAuthError, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data, actions } = useRecoveryFlow();
  const initialLocale = resolveUnauthenticatedLocale();
  const i18n = useI18n(initialLocale);

  const errorMessage = $derived(formatAuthError(i18n, data.state.error, initialLocale));

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    if ((await actions.submitRecoveryRequest()) === '/login/recovery/sent') {
      await goto('/login/recovery/sent');
    }
  }

  function handleEmailInput(event: Event) {
    const target = event.target as HTMLInputElement;
    actions.setEmail(target.value);
  }
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <h1 class="auth-shell__heading">{i18n.t('common.recoveryTitle')}</h1>
      <p class="auth-shell__body">{i18n.t('common.recoveryDescription')}</p>
    </div>

    {#if errorMessage}
      <p class="auth-shell__error" role="alert">{errorMessage}</p>
    {/if}

    <form class="auth-shell__form" onsubmit={handleSubmit}>
      <div class="flex flex-col gap-1.5">
        <Label for="recovery-email">{i18n.t('common.recoveryEmailLabel')}</Label>
        <Input
          id="recovery-email"
          type="email"
          autocomplete="email"
          required
          placeholder={i18n.t('common.recoveryEmailPlaceholder')}
          value={data.state.email}
          oninput={handleEmailInput}
          disabled={data.state.phase === 'submitting'}
        />
      </div>

      <Button
        type="submit"
        size="lg"
        disabled={data.state.phase === 'submitting' || data.state.email.trim() === ''}
      >
        {#if data.state.phase === 'submitting'}
          {i18n.t('common.recoverySending')}
        {:else}
          {i18n.t('common.recoverySubmit')}
        {/if}
      </Button>
    </form>

    <a class="auth-shell__link justify-center" href="/login">
      {i18n.t('common.recoveryBackToLogin')}
    </a>
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('common.recoveryBackToPublic')}</a>
  {/snippet}
</AuthLayout>
