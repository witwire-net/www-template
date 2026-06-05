<script lang="ts">
  /**
   * はじめるページ。
   * テンプレート利用者向けの短い行リスト。
   * 冗長説明・装飾コピー・詩的コピーは禁止。
   */
  import { useAccount } from '@www-template/domain';
  import { Item } from '@www-template/ui/components';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  /** はじめるステップ。テンプレート利用者向けの最小アクション。 */
  const steps = [
    { key: 'step1', href: '/settings' },
    { key: 'step2', href: '/settings/sign-in' },
  ] as const;
</script>

<section class="flex flex-col gap-6 p-6">
  <header class="flex flex-col gap-2 border-b border-border pb-4">
    <h1 class="text-2xl font-bold">{i18n.t('common.getting-started.title')}</h1>
  </header>

  <Item.ItemGroup>
    {#each steps as step (step.key)}
      <Item.Item variant="outline" size="sm">
        <Item.ItemHeader>
          <Item.ItemContent>
            <a href={step.href} class="text-sm font-medium hover:underline">
              {i18n.t(`common.getting-started.${step.key}`)}
            </a>
          </Item.ItemContent>
        </Item.ItemHeader>
      </Item.Item>
    {/each}
  </Item.ItemGroup>
</section>
