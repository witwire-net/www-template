<script lang="ts">
	import { Badge, CardNS, Table } from '@www-template/ui/components';

	const { data } = $props<{
		data: {
			stats: { totalAccounts: number; activeAccounts: number; suspendedAccounts: number; recentAccounts: { id: string; email: string; status: string; createdAt: Date }[] };
			recentAudit: { id: string; action: string; targetType: string; targetId: string; outcome: string; createdAt: Date }[];
		};
	}>();

	const kpis = $derived([
		{ label: 'Total accounts', value: data.stats.totalAccounts, tone: 'text-slate-950' },
		{ label: 'Active accounts', value: data.stats.activeAccounts, tone: 'text-emerald-700' },
		{ label: 'Suspended accounts', value: data.stats.suspendedAccounts, tone: 'text-rose-700' },
	]);

	function formatDate(value: Date): string {
		// DB 由来の Date を管理者が監査しやすい ISO 形式へ固定する。
		return new Date(value).toISOString();
	}
</script>

<main class="space-y-8 p-8">
	<section class="space-y-2">
		<p class="text-sm font-semibold uppercase tracking-wide text-slate-500">Command overview</p>
		<h1 class="text-3xl font-bold tracking-tight">Admin Console</h1>
		<p class="max-w-2xl text-slate-600">顧客アカウントの状態と直近の監査イベントを確認します。</p>
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
				<CardNS.CardTitle>Recent accounts</CardNS.CardTitle>
				<CardNS.CardDescription>新規作成されたアカウントの最新 5 件です。</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent>
				<Table.Table>
					<Table.TableHeader>
						<Table.TableRow>
							<Table.TableHead>Email</Table.TableHead>
							<Table.TableHead>Status</Table.TableHead>
							<Table.TableHead>Created</Table.TableHead>
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
				<CardNS.CardTitle>Recent audit</CardNS.CardTitle>
				<CardNS.CardDescription>直近の管理操作を outcome とともに確認します。</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent>
				<Table.Table>
					<Table.TableHeader>
						<Table.TableRow>
							<Table.TableHead>Action</Table.TableHead>
							<Table.TableHead>Target</Table.TableHead>
							<Table.TableHead>Outcome</Table.TableHead>
							<Table.TableHead>Created</Table.TableHead>
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
