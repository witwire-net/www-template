<script lang="ts">
  /**
   * 設定ページ。
   * Card ラッパーを廃止し、section/row ベースの構成にする。
   * 表示言語の設定とログインと端末への導線を提供する。
   */
  import { useAccount } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import * as Select from '@www-template/ui/components/select';
  import { Button } from '@www-template/ui/components';
  import {
    SUPPORTED_LOCALES,
    persistAppLocale,
    resolveUnauthenticatedLocale,
    useI18n,
    type Locale,
  } from '$lib/i18n';

  const { data: accountData, actions: accountActions } = useAccount();
  const { actions: sessionActions } = useAuthSession();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  let saving = $state(false);
  let success = $state(false);

  async function handleLocaleChange(value: string): Promise<void> {
    if (!SUPPORTED_LOCALES.includes(value as Locale)) {
      return;
    }
    const locale = value as Locale;
    if (locale !== accountData.state.account?.setting.locale) {
      saving = true;
      success = false;
      const headers = sessionActions.createAuthorizationHeaders();
      const ok = await accountActions.updateLocale(locale, headers);
      if (ok) {
        persistAppLocale(locale);
        success = true;
      }
      saving = false;
    }
  }
</script>

<section class="flex flex-col gap-6 p-6">
  <header class="flex flex-col gap-2 border-b border-border pb-4">
    <h1 class="text-2xl font-bold">{i18n.t('settings.title')}</h1>
  </header>

  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <label for="locale-select" class="text-sm font-medium">
        {i18n.t('settings.localeLabel')}
      </label>
      <Select.Root
        type="single"
        value={locale}
        onValueChange={(value: string) => {
          void handleLocaleChange(value);
        }}
      >
        <Select.Trigger id="locale-select">
          <Select.Value />
        </Select.Trigger>
        <Select.Content>
          <Select.Item value="ja">{i18n.t('settings.jaLabel')}</Select.Item>
          <Select.Item value="en">{i18n.t('settings.enLabel')}</Select.Item>
        </Select.Content>
      </Select.Root>
    </div>

    {#if saving}
      <p class="text-sm text-muted-foreground" aria-live="polite">
        {i18n.t('settings.saving')}
      </p>
    {/if}

    {#if success}
      <p class="text-sm text-success" aria-live="polite">
        {i18n.t('settings.success')}
      </p>
    {/if}

    {#if accountData.state.error !== null}
      <p class="text-sm text-destructive" role="alert">
        {i18n.t('settings.error')}
      </p>
    {/if}
  </div>

  <div class="flex flex-col gap-2 border-t border-border pt-4">
    <Button variant="outline" href="/settings/sign-in">
      {i18n.t('settings.signInAndDevices')}
    </Button>
  </div>
</section>
