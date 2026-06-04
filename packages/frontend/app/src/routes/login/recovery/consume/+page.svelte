<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { AuthPanel, Button, StatusIcon } from '@www-template/ui';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
  import { removeQueryParamFromUrl } from '../../../../lib/auth/url';
  import { formatAuthError, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data, actions } = useRecoveryFlow();
  const initialLocale = resolveUnauthenticatedLocale();
  const i18n = useI18n(initialLocale);

  const errorMessage = $derived(formatAuthError(i18n, data.state.error, initialLocale));

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
  <AuthPanel width="narrow">
    {#if consumePhase === 'done'}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="check" tone="accent" />
        <h1 class="auth-shell__heading">{i18n.t('common.recoveryConsumeDoneTitle')}</h1>
        <p class="auth-shell__body">{i18n.t('common.recoveryConsumeDoneDescription')}</p>
      </div>

      <Button size="lg" onclick={goToRegisterPasskey}>
        {i18n.t('common.recoveryConsumeDoneButton')}
      </Button>
    {:else if data.state.phase === 'invalid'}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="alert-circle" tone="destructive" />
        <h1 class="auth-shell__heading">{i18n.t('common.recoveryConsumeInvalidTitle')}</h1>
        <p class="auth-shell__body">
          {errorMessage || i18n.t('common.recoveryConsumeInvalidDescription')}
        </p>
      </div>

      <a class="auth-shell__link justify-center" href="/login/recovery">
        {i18n.t('common.recoveryConsumeRetry')}
      </a>
    {:else}
      <div class="flex flex-col items-center gap-3 text-center">
        <StatusIcon name="loader" tone="accent" />
        <h1 class="auth-shell__heading">{i18n.t('common.recoveryConsumeCheckingTitle')}</h1>
        <p class="auth-shell__body">{i18n.t('common.recoveryConsumeCheckingDescription')}</p>
      </div>
    {/if}
  </AuthPanel>

  {#snippet footer()}
    <a class="auth-shell__link" href="/">{i18n.t('common.recoveryConsumeBackToPublic')}</a>
  {/snippet}
</AuthLayout>
