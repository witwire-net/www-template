<script lang="ts">
	import { Badge, Button, CodeBlock, Table } from '@www-template/ui/components';

	import PaginationFooter from '$lib/components/shared/PaginationFooter.svelte';

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
		labels,
	}: {
		events: AuditEvent[];
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels: Record<string, string>;
	} = $props();

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

<PaginationFooter {page} {totalPages} perPage={25} {onPageChange} labels={labels} />
