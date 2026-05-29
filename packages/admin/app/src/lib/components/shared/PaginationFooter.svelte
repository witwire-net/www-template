<script lang="ts">
	import { Pagination } from '@www-template/ui/components';

	const {
		page = 1,
		totalPages = 1,
		perPage = 10,
		onPageChange,
		labels,
	}: {
		page?: number;
		totalPages?: number;
		perPage?: number;
		onPageChange?: (page: number) => void;
		labels: Record<string, string>;
	} = $props();
</script>

{#if totalPages > 1}
	<Pagination.Pagination
		aria-label={labels.pagination}
		count={totalPages * perPage}
		{perPage}
		{page}
		onPageChange={(p: number) => { onPageChange?.(p); }}
	>
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
