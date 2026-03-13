<svelte:options runes={true} />

<script lang="ts">
  import Pagination from './Pagination.svelte';

  let page = $state(2);
</script>

<Pagination
  {page}
  pageCount={5}
  ariaLabel="Results pages"
  previousLabel="Previous"
  previousAriaLabel="Go to previous page"
  nextLabel="Next"
  nextAriaLabel="Go to next page"
  pageAriaLabel={(pageNumber, isCurrent) =>
    isCurrent ? `Current page, page ${pageNumber}` : `Go to page ${pageNumber}`}
  onPageChange={(nextPage) => (page = nextPage)}
/>
