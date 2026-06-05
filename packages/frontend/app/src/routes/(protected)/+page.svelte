<script lang="ts">
  /**
   * ホームページ。
   * ダミー統計・旧アクションリンクを廃止し、
   * セクション＋行ベースのシンプルな構成にする。
   */
  import { useAccount } from '@www-template/domain';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));
</script>

<section class="flex flex-col gap-6 p-6">
  <header class="flex flex-col gap-2 border-b border-border pb-4">
    <h1 class="text-2xl font-bold">{i18n.t('common.home.title')}</h1>
  </header>
</section>
