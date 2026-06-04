<script lang="ts">
  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { AuthPanel, MonoLabel, StatusIcon } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data } = useRecoveryFlow();
  const i18n = useI18n(resolveUnauthenticatedLocale());
</script>

<AuthLayout>
  <AuthPanel width="narrow">
    <div class="flex flex-col items-center gap-3 text-center">
      <StatusIcon name="check" tone="success" />
      <h1 class="auth-shell__heading">{data.state.sentView.title}</h1>
      <p class="auth-shell__body">{data.state.sentView.description}</p>
      <MonoLabel tone="muted">{data.state.sentView.helper}</MonoLabel>
    </div>

    <a class="auth-shell__link justify-center" href="/login">
      {i18n.t('common.recoverySentBackToLogin')}
    </a>
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('common.recoverySentBackToPublic')}</a>
  {/snippet}
</AuthLayout>
