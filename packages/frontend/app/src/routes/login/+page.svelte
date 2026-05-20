<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { usePasskeyLogin } from '@www-template/domain/auth/passkey';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
  import { persistAppLocale, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data, actions } = usePasskeyLogin();

  // 認証前 fallback locale で i18n を初期化する
  const initialLocale = resolveUnauthenticatedLocale();
  persistAppLocale(initialLocale);
  const i18n = useI18n(initialLocale);

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
  <Card class="w-full">
    <CardContent>
      <div class="flex flex-col items-center gap-4 text-center">
        <h1 class="m-0 text-2xl font-bold text-center">
          {i18n.t('login.login')}
        </h1>
        <p class="m-0 text-sm text-muted-foreground text-center">
          {i18n.t('login.passkeyPrompt')}
        </p>

        {#if data.state.error}
          <p class="text-destructive text-sm m-0" role="alert">{data.state.error}</p>
        {/if}

        <Button
          class="w-full"
          type="button"
          disabled={data.state.isSubmitting}
          onclick={handlePasskeySignIn}
        >
          {#if data.state.isSubmitting}
            {i18n.t('login.signingIn')}
          {:else}
            {i18n.t('login.signInButton')}
          {/if}
        </Button>

        <Separator />

        <a href="/login/recovery" class="text-sm text-muted-foreground no-underline hover:underline">
          {i18n.t('login.lostPasskey')}
        </a>
      </div>
    </CardContent>
  </Card>

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">
      {i18n.t('login.backToPublic')}
    </a>
  {/snippet}
</AuthLayout>
