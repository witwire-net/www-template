<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  import { removeQueryParamFromUrl } from '../../../../lib/auth/url';

  const { data, actions } = useRecoveryFlow();
  const i18n = useI18n(resolveUnauthenticatedLocale());

  /**
   * consume 完了後の表示フェーズ。
   * - 'idle': 初期状態 / consuming 前
   * - 'done': device-link token の consume が成功し、デバイスリンク用の案内を表示中
   */
  let consumePhase = $state<'idle' | 'done'>('idle');

  /** URL から token を取得し consume する。 */
  async function consumeTokenFromUrl() {
    const token = removeQueryParamFromUrl('token');

    if (token === null || token === '') {
      await goto('/login/recovery');
      return;
    }

    const result = await actions.consumeToken(token);
    if (result === null) {
      return;
    }

    if (result.kind === 'device-link') {
      /*
       * デバイスリンク用の token が確認できた場合は、同一ページでデバイスリンク用の
       * 完了案内を表示する。パスキー登録画面への遷移はユーザー操作に委ねる。
       */
      consumePhase = 'done';
      return;
    }

    if (result.path === '/login/recovery/register') {
      /*
       * consume → register は SvelteKit client-side routing で同一 module instance の
       * domain singleton state を共有する。sessionStorage には recovery secret を保存しない。
       */
      await goto('/login/recovery/register');
    } else if (result.path === '/login/recovery') {
      /* 画面に retry guidance を表示するのでそのまま留まる */
    }
  }

  /** デバイスリンク確認後にパスキー登録画面へ移動する。 */
  async function goToRegisterPasskey() {
    await goto('/login/recovery/register');
  }

  /* mount 時に token consume を実行 */
  void consumeTokenFromUrl();
</script>

<AuthLayout>
  <Card class="w-full">
    <CardContent>
      {#if consumePhase === 'done'}
        <!--
          デバイスリンク用 token の consume が成功した場合の案内。
          recovery とは異なり、すぐに登録画面へ遷移せずユーザー操作を待つ。
        -->
        <div class="flex flex-col items-center gap-4 text-center" role="region" aria-label={i18n.t('common.recoveryConsumeDoneTitle')}>
          <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.recoveryConsumeDoneTitle')}</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">
            {i18n.t('common.recoveryConsumeDoneDescription')}
          </p>

          <Separator />

          <Button onclick={goToRegisterPasskey}>
            {i18n.t('common.recoveryConsumeDoneButton')}
          </Button>
        </div>
      {:else}
        <div class="flex flex-col items-center gap-4 text-center" role="region" aria-label={i18n.t('common.recoveryConsumeCheckingTitle')}>
          {#if data.state.phase === 'consuming'}
            <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.recoveryConsumeCheckingTitle')}</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">{i18n.t('common.recoveryConsumeCheckingDescription')}</p>
          {:else if data.state.phase === 'invalid'}
            <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.recoveryConsumeInvalidTitle')}</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">
              {data.state.error ?? i18n.t('common.recoveryConsumeInvalidDescription')}
            </p>

            <Separator />

            <a href="/login/recovery" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.recoveryConsumeRetry')}</a>
          {:else}
            <h1 class="m-0 text-2xl font-bold text-center">{i18n.t('common.recoveryConsumeCheckingTitle')}</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">{i18n.t('common.recoveryConsumeCheckingDescription')}</p>
          {/if}
        </div>
      {/if}
    </CardContent>
  </Card>

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">{i18n.t('common.recoveryConsumeBackToPublic')}</a>
  {/snippet}
</AuthLayout>
