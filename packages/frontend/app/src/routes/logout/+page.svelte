<script lang="ts">
  import { goto } from '$app/navigation';

  import { useAuthSession } from '@www-template/domain/auth/session';
  import { AuthPanel, Button, StatusIcon } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { actions } = useAuthSession();
  const i18n = useI18n(resolveUnauthenticatedLocale());

  let isLoggingOut = $state(true);
  let logoutError = $state<string | null>(null);

  /** mount 時に logout を実行する。browser 環境でのみ発火。 */
  if (typeof window !== 'undefined') {
    void performLogout();
  }

  async function performLogout() {
    try {
      const intent = await actions.logoutCurrentSession();
      // 残りセッションがある場合は intent が null となり認証状態を維持する
      await goto(intent ?? '/');
    } catch {
      logoutError = i18n.t('common.logoutFailedTitle');
      isLoggingOut = false;
      /* fail-safe: state 消去して login へ */
      actions.clearInMemorySession();
      await goto('/login');
    }
  }
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    {#if isLoggingOut}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="loader" tone="accent" />
        <h1 class="auth-shell__heading">{i18n.t('common.logoutInProgressTitle')}</h1>
        <p class="auth-shell__body">{i18n.t('common.logoutInProgressDescription')}</p>
      </div>
    {:else if logoutError}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="alert-circle" tone="destructive" />
        <h1 class="auth-shell__heading">{i18n.t('common.logoutFailedTitle')}</h1>
        <p class="auth-shell__error" role="alert">{logoutError}</p>
      </div>

      <Button
        variant="outline"
        size="lg"
        onclick={() => {
          void goto('/login');
        }}
      >
        {i18n.t('common.logoutFailedButton')}
      </Button>
    {:else}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="check" tone="success" />
        <h1 class="auth-shell__heading">{i18n.t('common.logoutSuccessTitle')}</h1>
      </div>

      <Button
        variant="outline"
        size="lg"
        onclick={() => {
          void goto('/login');
        }}
      >
        {i18n.t('common.logoutSuccessButton')}
      </Button>
    {/if}
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('common.logoutBackToPublic')}</a>
  {/snippet}
</AuthLayout>
