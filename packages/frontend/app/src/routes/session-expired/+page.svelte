<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const i18n = useI18n(resolveUnauthenticatedLocale());
</script>

<AuthLayout>
  <Card class="w-full">
    <CardContent class="flex flex-col items-center gap-4 text-center">
      <span class="inline-flex text-warning" aria-hidden="true">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
          <path d="M12 9v4"/>
          <path d="M12 17h.01"/>
        </svg>
      </span>
      <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.sessionExpiredTitle')}</h1>
      <p class="m-0 text-sm text-muted-foreground text-center">{i18n.t('common.sessionExpiredDescription')}</p>
      <Button class="w-full" onclick={() => { void goto('/login'); }}>
        {i18n.t('common.sessionExpiredLoginButton')}
      </Button>
      <Separator />
      <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.sessionExpiredPublicLink')}</a>
    </CardContent>
  </Card>
</AuthLayout>
