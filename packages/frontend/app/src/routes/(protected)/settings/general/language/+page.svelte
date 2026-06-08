<script lang="ts">
  /**
   * 表示言語設定ページ。
   * 1ページ1概念: 表示言語の選択のみを担当する。
   * Select は必ず現在値を表示し、空表示をなくす。
   * 保存中/成功/失敗を aria-live で通知する。
   */
  import { useAccount } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import * as Select from '@www-template/ui/components/select';
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
  let error = $state(false);

  /** 現在の locale 表示ラベル。 */
  const currentLocaleLabel = $derived(
    locale === 'ja' ? i18n.t('settings.jaLabel') : i18n.t('settings.enLabel')
  );

  async function handleLocaleChange(value: string): Promise<void> {
    if (!SUPPORTED_LOCALES.includes(value as Locale)) {
      return;
    }
    const nextLocale = value as Locale;
    if (nextLocale !== accountData.state.account?.setting.locale) {
      saving = true;
      success = false;
      error = false;
      const headers = sessionActions.createAuthorizationHeaders();
      const ok = await accountActions.updateLocale(nextLocale, headers);
      if (ok) {
        persistAppLocale(nextLocale);
        success = true;
      } else {
        error = true;
      }
      saving = false;
    }
  }
</script>

<section class="flex flex-col gap-4">
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
      <Select.Trigger id="locale-select" class="w-48">
        <Select.Value placeholder={currentLocaleLabel} />
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

  {#if error}
    <p class="text-sm text-destructive" role="alert">
      {i18n.t('settings.error')}
    </p>
  {/if}
</section>
