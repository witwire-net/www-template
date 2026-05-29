<script lang="ts">
	import { Badge, CardNS, Table } from '@www-template/ui/components';

	import { createCurrentAdminI18n } from '$lib/i18n';

	interface DashboardData {
		stats: { totalAccounts: number; activeAccounts: number; suspendedAccounts: number; recentAccounts: { id: string; email: string; status: string; createdAt: Date }[] };
		recentAudit: { id: string; action: string; targetType: string; targetId: string; outcome: string; createdAt: Date }[];
	}

	const { data } = $props<{ data?: Partial<DashboardData> }>();
	const pageData = $derived<DashboardData>({
		stats: data?.stats ?? { totalAccounts: 0, activeAccounts: 0, suspendedAccounts: 0, recentAccounts: [] },
		recentAudit: data?.recentAudit ?? [],
	});
	const i18n = $derived(createCurrentAdminI18n());

	const kpis = $derived([
		{ label: i18n.t('dashboard.totalAccounts'), value: i18n.formatNumber(pageData.stats.totalAccounts), tone: 'text-foreground' },
		{ label: i18n.t('dashboard.activeAccounts'), value: i18n.formatNumber(pageData.stats.activeAccounts), tone: 'text-success' },
		{ label: i18n.t('dashboard.suspendedAccounts'), value: i18n.formatNumber(pageData.stats.suspendedAccounts), tone: 'text-error' },
	]);

	function formatDate(value: Date): string {
		// Admin locale state に従って日時を整形し、ISO の技術表現をそのまま画面へ出さない。
		return i18n.formatDateTime(value);
	}

	function accountStatusLabel(statusValue: string): string {
		// dashboard の recent account でも一覧画面と同じ status 表示を使う。
		if (statusValue === 'active') return i18n.t('accounts.active');
		if (statusValue === 'suspended') return i18n.t('accounts.suspended');
		return statusValue;
	}
</script>

<svelte:head>
	<title>{i18n.t('dashboard.title')}</title>
</svelte:head>

<main class="space-y-8 p-8">
	<section class="space-y-2">
		<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('dashboard.eyebrow')}</p>
		<h1 class="text-3xl font-bold tracking-tight">{i18n.t('dashboard.title')}</h1>
		<p class="max-w-2xl text-muted-foreground">{i18n.t('dashboard.description')}</p>
	</section>

	<section class="grid gap-4 md:grid-cols-3">
		{#each kpis as kpi (kpi.label)}
			<CardNS.Card>
				<CardNS.CardHeader>
					<CardNS.CardDescription>{kpi.label}</CardNS.CardDescription>
					<CardNS.CardTitle class={`text-4xl ${kpi.tone}`}>{kpi.value}</CardNS.CardTitle>
				</CardNS.CardHeader>
			</CardNS.Card>
		{/each}
	</section>

	<section class="grid gap-6 xl:grid-cols-2">
		<CardNS.Card>
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('dashboard.recentAccounts')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('dashboard.recentAccountsDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent>
				<Table.Table>
					<Table.TableHeader>
						<Table.TableRow>
							<Table.TableHead>{i18n.t('accounts.email')}</Table.TableHead>
							<Table.TableHead>{i18n.t('accounts.status')}</Table.TableHead>
							<Table.TableHead>{i18n.t('accounts.created')}</Table.TableHead>
						</Table.TableRow>
					</Table.TableHeader>
					<Table.TableBody>
						{#each pageData.stats.recentAccounts as account (account.id)}
							<Table.TableRow>
								<Table.TableCell>{account.email}</Table.TableCell>
								<Table.TableCell><Badge variant={account.status === 'active' ? 'success' : 'danger'}>{accountStatusLabel(account.status)}</Badge></Table.TableCell>
								<Table.TableCell>{formatDate(account.createdAt)}</Table.TableCell>
							</Table.TableRow>
						{/each}
					</Table.TableBody>
				</Table.Table>
			</CardNS.CardContent>
		</CardNS.Card>

		<CardNS.Card>
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('dashboard.recentAudit')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('dashboard.recentAuditDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent>
				<Table.Table>
					<Table.TableHeader>
						<Table.TableRow>
							<Table.TableHead>{i18n.t('audit.action')}</Table.TableHead>
							<Table.TableHead>{i18n.t('audit.target')}</Table.TableHead>
							<Table.TableHead>{i18n.t('audit.outcome')}</Table.TableHead>
							<Table.TableHead>{i18n.t('accounts.created')}</Table.TableHead>
						</Table.TableRow>
					</Table.TableHeader>
					<Table.TableBody>
						{#each pageData.recentAudit as event (event.id)}
							<Table.TableRow>
								<Table.TableCell>{event.action}</Table.TableCell>
								<Table.TableCell>{event.targetType} {event.targetId}</Table.TableCell>
								<Table.TableCell><Badge variant={event.outcome === 'succeeded' ? 'success' : event.outcome === 'failed' ? 'danger' : 'warning'}>{event.outcome}</Badge></Table.TableCell>
								<Table.TableCell>{formatDate(event.createdAt)}</Table.TableCell>
							</Table.TableRow>
						{/each}
					</Table.TableBody>
				</Table.Table>
			</CardNS.CardContent>
		</CardNS.Card>
	</section>
</main>
