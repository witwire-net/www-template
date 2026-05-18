<script lang="ts">
	import { Badge, Table, Button, Pagination } from '@www-template/ui/components';

	interface AccountSummary {
		id: string;
		email: string;
		status: string;
		created_at: string;
	}

	const {
		accounts,
		onSelect,
		page = 1,
		totalPages = 1,
		onPageChange,
	}: {
		accounts: AccountSummary[];
		onSelect?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
	} = $props();
</script>

<Table.Table>
	<Table.TableCaption>Account list</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>Email</Table.TableHead>
			<Table.TableHead>Status</Table.TableHead>
			<Table.TableHead>Created</Table.TableHead>
			<Table.TableHead>Actions</Table.TableHead>
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#each accounts as account (account.id)}
			<Table.TableRow class="cursor-pointer" onclick={() => onSelect?.(account.id)}>
				<Table.TableCell>{account.email}</Table.TableCell>
				<Table.TableCell>
					<Badge variant={account.status === 'active' ? 'success' : 'danger'}>
						{account.status}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>{account.created_at}</Table.TableCell>
				<Table.TableCell>
					<Button variant="outline" size="xs">View</Button>
				</Table.TableCell>
			</Table.TableRow>
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
