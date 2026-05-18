<script lang="ts" generics="T">
	import { EmptyState, Button, Table } from '@www-template/ui/components';
	import { Skeleton } from '@www-template/ui/components/skeleton';

	import type { Snippet } from 'svelte';

	interface Column {
		key: string;
		label: string;
		sortable?: boolean;
		render?: Snippet<[T]>;
	}

	const {
		columns,
		rows,
		loading = false,
		emptyMessage = 'データがありません',
		onRowClick,
	}: {
		columns: Column[];
		rows: T[];
		loading?: boolean;
		emptyMessage?: string;
		onRowClick?: (row: T) => void;
	} = $props();

	let sortKey = $state<string | null>(null);
	let sortOrder = $state<'asc' | 'desc'>('asc');

	function toggleSort(key: string): void {
		if (sortKey === key) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortOrder = 'asc';
		}
	}

	function getValue(row: T, key: string): unknown {
		return Reflect.get(row as Record<string, unknown>, key);
	}

	function formatCell(value: unknown): string {
		if (value == null) return '';
		if (typeof value === 'string') return value;
		if (typeof value === 'number' || typeof value === 'boolean') return String(value);
		try {
			return JSON.stringify(value);
		} catch {
			return '';
		}
	}

	const sortedRows = $derived.by(() => {
		if (sortKey === null) return rows;
		const key = sortKey;
		const order = sortOrder;
		return [...rows].sort((a, b) => {
			const av = getValue(a, key);
			const bv = getValue(b, key);
			if (av === bv) return 0;
			if (av == null) return 1;
			if (bv == null) return -1;
			return (av < bv ? -1 : 1) * (order === 'asc' ? 1 : -1);
		});
	});
</script>

<Table.Table data-testid="data-table">
	<Table.TableCaption class="sr-only">Data table</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			{#each columns as col (col.key)}
				<Table.TableHead>
					{#if col.sortable}
						<Button variant="ghost" size="xs" onclick={() => { toggleSort(col.key); }}>
							{col.label}
							{sortKey === col.key ? (sortOrder === 'asc' ? '▲' : '▼') : '◦'}
						</Button>
					{:else}
						{col.label}
					{/if}
				</Table.TableHead>
			{/each}
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#if loading}
			{#each { length: 5 } as _, i (i)}
				<Table.TableRow>
					{#each columns as _, j (j)}
						<Table.TableCell><Skeleton class="h-4 w-3/4" /></Table.TableCell>
					{/each}
				</Table.TableRow>
			{/each}
		{:else if sortedRows.length === 0}
			<Table.TableRow>
				<Table.TableCell colspan={columns.length}>
					<EmptyState title={emptyMessage} />
				</Table.TableCell>
			</Table.TableRow>
		{:else}
			{#each sortedRows as row, i (i)}
				<Table.TableRow
					class={onRowClick != null ? 'cursor-pointer' : ''}
					onclick={() => { onRowClick?.(row); }}
				>
					{#each columns as col (col.key)}
						<Table.TableCell>
							{#if col.render}
								{@render col.render(row)}
							{:else}
								{formatCell(getValue(row, col.key))}
							{/if}
						</Table.TableCell>
					{/each}
				</Table.TableRow>
			{/each}
		{/if}
	</Table.TableBody>
</Table.Table>
