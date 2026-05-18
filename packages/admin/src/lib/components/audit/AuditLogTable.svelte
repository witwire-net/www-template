<script lang="ts">
	import { Badge, Button, CodeBlock, Pagination, Table } from '@www-template/ui/components';

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
	}: {
		events: AuditEvent[];
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
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
	<Table.TableCaption>Audit log</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>Timestamp</Table.TableHead>
			<Table.TableHead>Operator</Table.TableHead>
			<Table.TableHead>Action</Table.TableHead>
			<Table.TableHead>Target</Table.TableHead>
			<Table.TableHead>Outcome</Table.TableHead>
			<Table.TableHead>Details</Table.TableHead>
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
							{expandedId === event.id ? 'Hide' : 'Show'}
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
	<Pagination.Pagination count={totalPages * 10} perPage={10} {page} onPageChange={(p: number) => { onPageChange?.(p); }}>
		<Pagination.PaginationContent>
			<Pagination.PaginationItem>
				<Pagination.PaginationPrevButton />
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationLink page={{ type: 'page', value: page }} isActive>
					{page}
				</Pagination.PaginationLink>
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationNextButton />
			</Pagination.PaginationItem>
		</Pagination.PaginationContent>
	</Pagination.Pagination>
{/if}
