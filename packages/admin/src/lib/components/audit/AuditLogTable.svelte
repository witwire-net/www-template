<script lang="ts">
	import { Badge, Button, CodeBlock, Pagination, Table } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	interface AuditEvent {
		id: string;
		operator_email?: string;
		action: string;
		target_type?: string;
		target_id?: string;
		outcome: string;
		details?: Record<string, unknown>;
		created_at: string;
	}

	const {
		events,
		page = 1,
		totalPages = 1,
		onPageChange,
		labels = createAuditTableLabels(),
	}: {
		events: AuditEvent[];
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels?: ReturnType<typeof createAuditTableLabels>;
	} = $props();

	function createAuditTableLabels() {
		// 監査 table component 単体利用時も Admin-owned fallback 辞書で文言を取得する。
		const { t } = createAdminI18n();
		return {
			caption: t('audit.tableCaption'),
			timestamp: t('audit.timestamp'),
			operator: t('audit.operator'),
			action: t('audit.action'),
			target: t('audit.target'),
			outcome: t('audit.outcome'),
			details: t('audit.details'),
			show: t('audit.show'),
			hide: t('audit.hide'),
			pagination: t('shared.pagination'),
			previousPage: t('shared.previousPage'),
			nextPage: t('shared.nextPage'),
		};
	}

	let expandedId = $state<string | null>(null);

	function toggleExpand(id: string): void {
		expandedId = expandedId === id ? null : id;
	}

	function stringifyDetails(details: Record<string, unknown>): string {
		return JSON.stringify(details, null, 2);
	}
</script>

<Table.Table>
	<Table.TableCaption>{labels.caption}</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>{labels.timestamp}</Table.TableHead>
			<Table.TableHead>{labels.operator}</Table.TableHead>
			<Table.TableHead>{labels.action}</Table.TableHead>
			<Table.TableHead>{labels.target}</Table.TableHead>
			<Table.TableHead>{labels.outcome}</Table.TableHead>
			<Table.TableHead>{labels.details}</Table.TableHead>
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#each events as event (event.id)}
			<Table.TableRow>
				<Table.TableCell>{event.created_at}</Table.TableCell>
				<Table.TableCell>{event.operator_email ?? '-'}</Table.TableCell>
				<Table.TableCell>{event.action}</Table.TableCell>
				<Table.TableCell>{(event.target_type ?? '') + ' ' + (event.target_id ?? '')}</Table.TableCell>
				<Table.TableCell>
					<Badge variant={event.outcome === 'succeeded' ? 'success' : event.outcome === 'failed' ? 'danger' : 'warning'}>
						{event.outcome}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>
					{#if event.details != null}
						<Button variant="ghost" size="xs" onclick={() => { toggleExpand(event.id); }}>
						{expandedId === event.id ? labels.hide : labels.show}
						</Button>
					{/if}
				</Table.TableCell>
			</Table.TableRow>
			{#if expandedId === event.id && event.details != null}
				<Table.TableRow>
					<Table.TableCell colspan={6}>
						<CodeBlock value={stringifyDetails(event.details)} />
					</Table.TableCell>
				</Table.TableRow>
			{/if}
		{/each}
	</Table.TableBody>
</Table.Table>

{#if totalPages > 1}
	<Pagination.Pagination aria-label={labels.pagination} count={totalPages * 10} perPage={10} {page} onPageChange={(p: number) => { onPageChange?.(p); }}>
		<Pagination.PaginationContent>
			<Pagination.PaginationItem>
				<Pagination.PaginationPrevButton aria-label={labels.previousPage} />
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationLink page={{ type: 'page', value: page }} isActive>
					{page}
				</Pagination.PaginationLink>
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationNextButton aria-label={labels.nextPage} />
			</Pagination.PaginationItem>
		</Pagination.PaginationContent>
	</Pagination.Pagination>
{/if}
