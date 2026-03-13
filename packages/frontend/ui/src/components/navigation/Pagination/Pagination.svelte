<svelte:options runes={true} />

<script lang="ts">
  import { joinClassNames } from '@ui/components/navigation/shared';

  import styles from './Pagination.module.scss';

  type PageAriaLabel = (pageNumber: number, isCurrent: boolean) => string;

  type Props = {
    ariaLabel?: string;
    nextAriaLabel?: string;
    nextLabel?: string;
    onPageChange: (page: number) => void;
    page: number;
    pageAriaLabel?: PageAriaLabel;
    pageCount: number;
    previousAriaLabel?: string;
    previousLabel?: string;
  };

  function defaultPageAriaLabel(pageNumber: number, isCurrent: boolean): string {
    return isCurrent ? `Current page, page ${pageNumber}` : `Go to page ${pageNumber}`;
  }

  let {
    ariaLabel = 'Pagination',
    nextAriaLabel = undefined,
    nextLabel = 'Next',
    onPageChange,
    page,
    pageAriaLabel = defaultPageAriaLabel,
    pageCount,
    previousAriaLabel = undefined,
    previousLabel = 'Prev',
  }: Props = $props();

  const pages = $derived(Array.from({ length: pageCount }, (_, index) => index + 1));
  const resolvedNextAriaLabel = $derived(nextAriaLabel ?? nextLabel);
  const resolvedPreviousAriaLabel = $derived(previousAriaLabel ?? previousLabel);

  function getPageClassName(pageNumber: number): string {
    return joinClassNames(styles.page ?? '', pageNumber === page ? (styles.active ?? '') : undefined);
  }
</script>

<nav class={styles.pagination ?? ''} aria-label={ariaLabel}>
  <button
    type="button"
    class={styles.button ?? ''}
    aria-label={resolvedPreviousAriaLabel}
    onclick={() => {
      onPageChange(Math.max(1, page - 1));
    }}
    disabled={page <= 1}
  >
    {previousLabel}
  </button>
  {#each pages as pageNumber (pageNumber)}
    <button
      type="button"
      class={getPageClassName(pageNumber)}
      aria-label={pageAriaLabel(pageNumber, pageNumber === page)}
      aria-current={pageNumber === page ? 'page' : undefined}
      onclick={() => {
        onPageChange(pageNumber);
      }}
    >
      {pageNumber}
    </button>
  {/each}
  <button
    type="button"
    class={styles.button ?? ''}
    aria-label={resolvedNextAriaLabel}
    onclick={() => {
      onPageChange(Math.min(pageCount, page + 1));
    }}
    disabled={page >= pageCount}
  >
    {nextLabel}
  </button>
</nav>
