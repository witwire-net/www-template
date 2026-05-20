<script lang="ts">
	import { CardNS, EmptyState } from '@www-template/ui/components';

	import { goto } from '$app/navigation';

	import AuditFilterBar from '$lib/components/audit/AuditFilterBar.svelte';
	import AuditLogTable from '$lib/components/audit/AuditLogTable.svelte';
	import { createAdminI18n } from '$lib/i18n';

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
		data: { locale: 'ja' | 'en'; events: AuditRow[]; operators: OperatorRow[]; page: number; totalPages: number; filters: Record<string, string> };
	}>();
	const i18n = $derived(createAdminI18n(data.locale));

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
		<h1 class="text-3xl font-bold tracking-tight">{i18n.t('audit.title')}</h1>
		<p class="text-slate-600">{i18n.t('audit.description')}</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>{i18n.t('audit.filters')}</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent><AuditFilterBar labels={{ operator: i18n.t('audit.operator'), all: i18n.t('audit.all'), action: i18n.t('audit.action'), actionPlaceholder: i18n.t('audit.actionPlaceholder'), from: i18n.t('audit.from'), to: i18n.t('audit.to'), filter: i18n.t('audit.filter'), clear: i18n.t('audit.clear') }} operators={data.operators} onFilter={(filters: Record<string, string | undefined>) => { applyFilters(filters); }} /></CardNS.CardContent>
	</CardNS.Card>
	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>{i18n.t('audit.events')}</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent>
			{#if events.length === 0}<EmptyState title={i18n.t('audit.emptyTitle')} description={i18n.t('audit.emptyDescription')} />{:else}<AuditLogTable labels={{ caption: i18n.t('audit.tableCaption'), timestamp: i18n.t('audit.timestamp'), operator: i18n.t('audit.operator'), action: i18n.t('audit.action'), target: i18n.t('audit.target'), outcome: i18n.t('audit.outcome'), details: i18n.t('audit.details'), show: i18n.t('audit.show'), hide: i18n.t('audit.hide'), pagination: i18n.t('shared.pagination'), previousPage: i18n.t('shared.previousPage'), nextPage: i18n.t('shared.nextPage') }} {events} page={data.page} totalPages={data.totalPages} onPageChange={(page: number) => { applyFilters(data.filters, page); }} />{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
