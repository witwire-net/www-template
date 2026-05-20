<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { Button, Card, CardContent } from '@www-template/ui/components';
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
  <Card class="w-full">
    <CardContent>
      <div class="flex flex-col items-center gap-4 text-center" role="region" aria-label={i18n.t('common.logoutRegionLabel')}>
        {#if isLoggingOut}
          <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.logoutInProgressTitle')}</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">{i18n.t('common.logoutInProgressDescription')}</p>
        {:else if logoutError}
          <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.logoutFailedTitle')}</h1>
          <p class="text-destructive text-sm m-0" role="alert">{logoutError}</p>
          <Button variant="secondary" class="w-full" onclick={() => { void goto('/login'); }}>
            {i18n.t('common.logoutFailedButton')}
          </Button>
        {:else}
          <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.logoutSuccessTitle')}</h1>
          <Button variant="secondary" class="w-full" onclick={() => { void goto('/login'); }}>
            {i18n.t('common.logoutSuccessButton')}
          </Button>
        {/if}
      </div>
    </CardContent>
  </Card>

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.logoutBackToPublic')}</a>
  {/snippet}
</AuthLayout>
