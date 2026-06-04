<script lang="ts">
  import { goto } from '$app/navigation';
  import { env } from '$env/dynamic/public';

  import { AuthPanel, Button, StatusIcon } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const supportHref = env.PUBLIC_SUPPORT_URL ?? '/login/recovery';
  const i18n = useI18n(resolveUnauthenticatedLocale());
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <StatusIcon name="shield-x" tone="destructive" />
      <h1 class="auth-shell__heading">{i18n.t('common.accountSuspendedTitle')}</h1>
      <p class="auth-shell__body">{i18n.t('common.accountSuspendedDescription1')}</p>
      <p class="auth-shell__body">{i18n.t('common.accountSuspendedDescription2')}</p>
    </div>

    <div class="auth-shell__actions">
      <Button
        size="lg"
        href={supportHref}
        target={supportHref.startsWith('http') ? '_blank' : undefined}
        rel={supportHref.startsWith('http') ? 'noopener noreferrer' : undefined}
      >
        {i18n.t('common.accountSuspendedSupportButton')}
      </Button>
      <Button
        variant="outline"
        size="lg"
        onclick={() => {
          void goto('/login');
        }}
      >
        {i18n.t('common.accountSuspendedAlternativeLogin')}
      </Button>
    </div>

    <a class="auth-shell__link justify-center" href="/">
      {i18n.t('common.accountSuspendedPublicLink')}
    </a>
  </AuthPanel>
</AuthLayout>
