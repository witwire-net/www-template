<script lang="ts">
	import { Badge, CardNS, Table } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	const { data } = $props<{
		data: {
			locale: 'ja' | 'en';
			stats: { totalAccounts: number; activeAccounts: number; suspendedAccounts: number; recentAccounts: { id: string; email: string; status: string; createdAt: Date }[] };
			recentAudit: { id: string; action: string; targetType: string; targetId: string; outcome: string; createdAt: Date }[];
		};
	}>();
	const i18n = $derived(createAdminI18n(data.locale));

	const kpis = $derived([
		{ label: i18n.t('dashboard.totalAccounts'), value: data.stats.totalAccounts, tone: 'text-slate-950' },
		{ label: i18n.t('dashboard.activeAccounts'), value: data.stats.activeAccounts, tone: 'text-emerald-700' },
		{ label: i18n.t('dashboard.suspendedAccounts'), value: data.stats.suspendedAccounts, tone: 'text-rose-700' },
	]);

	function formatDate(value: Date): string {
		// DB 由来の Date を管理者が監査しやすい ISO 形式へ固定する。
		return new Date(value).toISOString();
	}
</script>

<main class="space-y-8 p-8">
	<section class="space-y-2">
		<p class="text-sm font-semibold uppercase tracking-wide text-slate-500">{i18n.t('dashboard.eyebrow')}</p>
		<h1 class="text-3xl font-bold tracking-tight">{i18n.t('dashboard.title')}</h1>
		<p class="max-w-2xl text-slate-600">{i18n.t('dashboard.description')}</p>
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
						{#each data.stats.recentAccounts as account (account.id)}
							<Table.TableRow>
								<Table.TableCell>{account.email}</Table.TableCell>
								<Table.TableCell><Badge variant={account.status === 'active' ? 'success' : 'danger'}>{account.status}</Badge></Table.TableCell>
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
						{#each data.recentAudit as event (event.id)}
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
