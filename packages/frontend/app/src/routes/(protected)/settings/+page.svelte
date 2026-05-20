<script lang="ts">
  import { useAccount } from '@www-template/domain';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { Card, CardContent } from '@www-template/ui/components';
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

<Card class="w-full max-w-md">
  <CardContent>
    <div class="flex flex-col gap-4">
      <h1 class="text-2xl font-bold">
        {i18n.t('settings.title')}
      </h1>

      <div class="flex flex-col gap-2">
        <label for="locale-select" class="text-sm font-medium">
          {i18n.t('settings.localeLabel')}
        </label>
        <p class="text-sm text-muted-foreground">
          {i18n.t('settings.localeDescription')}
        </p>
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
        <p class="text-sm text-muted-foreground">
          {i18n.t('settings.saving')}
        </p>
      {/if}

      {#if success}
        <p class="text-sm text-success">
          {i18n.t('settings.success')}
        </p>
      {/if}

      {#if accountData.state.error !== null}
        <p class="text-sm text-destructive" role="alert">
          {i18n.t('settings.error')}
        </p>
      {/if}
    </div>
  </CardContent>
</Card>
