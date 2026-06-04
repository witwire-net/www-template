<script lang="ts">
  import { useAccount } from '@www-template/domain';
  import { MonoLabel } from '@www-template/ui';

  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));
</script>

<section class="overview-section">
  <header class="overview-section__head">
    <h2 class="overview-section__title">{i18n.t('common.overviewTitle')}</h2>
    <p class="overview-section__lead">{i18n.t('common.overviewLead')}</p>
  </header>

  <div class="overview-section__stats">
    <article class="overview-section__stat">
      <MonoLabel tone="muted">{i18n.t('common.statSurfaceLabel')}</MonoLabel>
      <span class="overview-section__stat-value">{i18n.t('common.statSurfaceValue')}</span>
    </article>
    <article class="overview-section__stat">
      <MonoLabel tone="muted">{i18n.t('common.statSessionLabel')}</MonoLabel>
      <span class="overview-section__stat-value">{i18n.t('common.statSessionValue')}</span>
    </article>
    <article class="overview-section__stat">
      <MonoLabel tone="muted">{i18n.t('common.statFormatLabel')}</MonoLabel>
      <span class="overview-section__stat-value">{i18n.t('common.statFormatValue')}</span>
    </article>
  </div>

  <div class="overview-section__actions">
    <a class="overview-section__action-link overview-section__action-link--primary" href="/passkeys">
      {i18n.t('common.openPasskeys')}
    </a>
    <a class="overview-section__action-link" href="/sessions">
      {i18n.t('common.openSessions')}
    </a>
    <a class="overview-section__action-link" href="/settings">
      {i18n.t('common.openSettings')}
    </a>
    <a class="overview-section__action-link" href="/logout">
      {i18n.t('common.openLogout')}
    </a>
  </div>
</section>
