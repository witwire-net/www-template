<script lang="ts">
	import { CardNS, EmptyState } from '@www-template/ui/components';

	import { goto } from '$app/navigation';

	import AuditFilterBar from '$lib/components/audit/AuditFilterBar.svelte';
	import AuditLogTable from '$lib/components/audit/AuditLogTable.svelte';

	interface AuditRow {
		id: string;
		operatorId: string;
		action: string;
		targetType: string;
		targetId: string;
		outcome: string;
		details: Record<string, unknown> | null;
		createdAt: Date;
	}

	interface OperatorRow {
		id: string;
		email: string;
	}

	const { data } = $props<{
		data: { events: AuditRow[]; operators: OperatorRow[]; page: number; totalPages: number; filters: Record<string, string> };
	}>();

	const events = $derived(data.events.map((event: AuditRow) => ({ id: event.id, operator_email: data.operators.find((operator: OperatorRow) => operator.id === event.operatorId)?.email, action: event.action, target_type: event.targetType, target_id: event.targetId, outcome: event.outcome, details: event.details ?? undefined, created_at: new Date(event.createdAt).toISOString() })));

	function applyFilters(filters: Record<string, string | undefined>, page = 1): void {
		// フィルター状態を query string に固定し、監査証跡の共有と再現を可能にする。
		const params: string[] = [];
		for (const [key, value] of Object.entries(filters)) {
			if (value !== undefined && value !== '') params.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`);
		}
		params.push(`page=${encodeURIComponent(String(page))}`);
		void goto(`/audit?${params.join('&')}`);
	}
</script>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">Audit Log</h1>
		<p class="text-slate-600">管理操作を operator・action・日付で絞り込みます。</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>Filters</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent><AuditFilterBar operators={data.operators} onFilter={(filters: Record<string, string | undefined>) => { applyFilters(filters); }} /></CardNS.CardContent>
	</CardNS.Card>
	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>Events</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent>
			{#if events.length === 0}<EmptyState title="No audit events" description="条件に一致する監査イベントはありません。" />{:else}<AuditLogTable {events} page={data.page} totalPages={data.totalPages} onPageChange={(page: number) => { applyFilters(data.filters, page); }} />{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
