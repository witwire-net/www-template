<script lang="ts">
  import { goto } from '$app/navigation';

  import { AuthPanel, Button, StatusIcon } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const i18n = useI18n(resolveUnauthenticatedLocale());
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <StatusIcon name="alert-circle" tone="warning" />
      <h1 class="auth-shell__heading">{i18n.t('common.sessionExpiredTitle')}</h1>
      <p class="auth-shell__body">{i18n.t('common.sessionExpiredDescription')}</p>
    </div>

    <Button
      size="lg"
      onclick={() => {
        void goto('/login');
      }}
    >
      {i18n.t('common.sessionExpiredLoginButton')}
    </Button>

    <a class="auth-shell__link justify-center" href="/">
      {i18n.t('common.sessionExpiredPublicLink')}
    </a>
  </AuthPanel>
</AuthLayout>
