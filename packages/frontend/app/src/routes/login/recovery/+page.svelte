<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { Button, Card, CardContent, Input, Label, Separator } from '@www-template/ui/components';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data, actions } = useRecoveryFlow();
  const i18n = useI18n(resolveUnauthenticatedLocale());

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
  <Card class="w-full">
    <CardContent>
      <div class="flex flex-col items-center gap-4 text-center">
        <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.recoveryTitle')}</h1>
        <p class="m-0 text-sm text-muted-foreground text-center">
          {i18n.t('common.recoveryDescription')}
        </p>

        {#if data.state.error}
          <p class="text-destructive text-sm m-0" role="alert">{data.state.error}</p>
        {/if}

        <form class="w-full flex flex-col gap-2" onsubmit={handleSubmit}>
          <div class="flex flex-col gap-1 text-left">
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
            class="w-full"
            type="submit"
            disabled={data.state.phase === 'submitting' || data.state.email.trim() === ''}
          >
            {#if data.state.phase === 'submitting'}
              {i18n.t('common.recoverySending')}
            {:else}
              {i18n.t('common.recoverySubmit')}
            {/if}
          </Button>
        </form>

        <Separator />

        <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.recoveryBackToLogin')}</a>
      </div>
    </CardContent>
  </Card>

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.recoveryBackToPublic')}</a>
  {/snippet}
</AuthLayout>
