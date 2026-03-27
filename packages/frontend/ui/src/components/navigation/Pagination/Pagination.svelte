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
    /**
     * 現在ページの前後に常に表示するページ数（デフォルト: 1）
     * 例: siblingCount=1 のとき current=5 なら [4, 5, 6] が表示される
     */
    siblingCount?: number;
  };

  const ELLIPSIS = '...' as const;

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
    siblingCount = 1,
  }: Props = $props();

  const resolvedNextAriaLabel = $derived(nextAriaLabel ?? nextLabel);
  const resolvedPreviousAriaLabel = $derived(previousAriaLabel ?? previousLabel);

  /**
   * 表示するページ番号と省略記号の配列を返す。
   * 例: pageCount=10, page=5, siblingCount=1
   *   → [1, '...', 4, 5, 6, '...', 10]
   */
  const pageItems = $derived(buildPageItems(page, pageCount, siblingCount));

  function buildPageItems(
    current: number,
    total: number,
    siblings: number
  ): (number | typeof ELLIPSIS)[] {
    // ページ数が少ない場合は全て表示
    const totalDisplayed = 2 * siblings + 5; // 両端2 + sibling両側 + current + 省略記号2

    if (total <= totalDisplayed) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }

    const leftSibling = Math.max(current - siblings, 1);
    const rightSibling = Math.min(current + siblings, total);

    const showLeftEllipsis = leftSibling > 2;
    const showRightEllipsis = rightSibling < total - 1;

    const result: (number | typeof ELLIPSIS)[] = [];

    // 先頭
    result.push(1);

    if (showLeftEllipsis) {
      result.push(ELLIPSIS);
    } else {
      // 省略なし：1 の次から leftSibling の前まで連続表示
      for (let i = 2; i < leftSibling; i++) {
        result.push(i);
      }
    }

    // 現在ページ周辺
    for (let i = leftSibling; i <= rightSibling; i++) {
      result.push(i);
    }

    if (showRightEllipsis) {
      result.push(ELLIPSIS);
    } else {
      // 省略なし：rightSibling の次から末尾の前まで連続表示
      for (let i = rightSibling + 1; i < total; i++) {
        result.push(i);
      }
    }

    // 末尾
    result.push(total);

    return result;
  }

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

  {#each pageItems as item, idx (typeof item === 'number' ? item : `ellipsis-${String(idx)}`)}
    {#if item === ELLIPSIS}
      <span class={styles.ellipsis ?? ''} aria-hidden="true">…</span>
    {:else}
      <button
        type="button"
        class={getPageClassName(item)}
        aria-label={pageAriaLabel(item, item === page)}
        aria-current={item === page ? 'page' : undefined}
        onclick={() => {
          onPageChange(item);
        }}
      >
        {item}
      </button>
    {/if}
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
