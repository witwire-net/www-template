<script lang="ts">
  import { useAccount } from '@www-template/domain';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));
</script>

<section class="overview-section">
  <div class="overview-section__eyebrow">{i18n.t('common.overviewEyebrow')}</div>
  <h2>{i18n.t('common.overviewTitle')}</h2>
  <p>{i18n.t('common.overviewDescription')}</p>
  <a href="/logout" class="overview-section__link">{i18n.t('common.openLogout')}</a>
</section>
