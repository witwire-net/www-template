<script lang="ts">
	import { Badge, Table, Button, Pagination } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

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
		labels = createAccountTableLabels(),
	}: {
		accounts: AccountSummary[];
		onSelect?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels?: ReturnType<typeof createAccountTableLabels>;
	} = $props();

	function createAccountTableLabels() {
		// 親 route が labels を渡さない利用でも、Admin-owned fallback 辞書から表示文言を作る。
		const { t } = createAdminI18n();
		return {
			caption: t('accounts.tableCaption'),
			email: t('accounts.email'),
			status: t('accounts.status'),
			created: t('accounts.created'),
			actions: t('accounts.actions'),
			view: t('accounts.view'),
			pagination: t('shared.pagination'),
			previousPage: t('shared.previousPage'),
			nextPage: t('shared.nextPage'),
		};
	}
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
						{account.status}
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
