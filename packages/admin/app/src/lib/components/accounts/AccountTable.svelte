<script lang="ts">
	import { Badge, Table, Button } from '@www-template/ui/components';

	import PaginationFooter from '$lib/components/shared/PaginationFooter.svelte';

	interface AccountSummary {
		id: string;
		email: string;
		status: string;
		status_label?: string;
		created_at: string;
	}

	const {
		accounts,
		onSelect,
		page = 1,
		totalPages = 1,
		onPageChange,
		labels,
	}: {
		accounts: AccountSummary[];
		onSelect?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels: Record<string, string>;
	} = $props();
</script>

<Table.Table>
	<Table.TableCaption>{labels.caption}</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>{labels.email}</Table.TableHead>
			<Table.TableHead>{labels.status}</Table.TableHead>
			<Table.TableHead>{labels.created}</Table.TableHead>
			<Table.TableHead>{labels.actions}</Table.TableHead>
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#each accounts as account (account.id)}
			<Table.TableRow class="cursor-pointer" onclick={() => onSelect?.(account.id)}>
				<Table.TableCell>{account.email}</Table.TableCell>
				<Table.TableCell>
					<Badge variant={account.status === 'active' ? 'success' : 'danger'}>
						{account.status_label ?? account.status}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>{account.created_at}</Table.TableCell>
				<Table.TableCell>
					<Button variant="outline" size="xs">{labels.view}</Button>
				</Table.TableCell>
			</Table.TableRow>
		{/each}
	</Table.TableBody>
</Table.Table>

<PaginationFooter {page} {totalPages} perPage={20} {onPageChange} labels={labels} />
