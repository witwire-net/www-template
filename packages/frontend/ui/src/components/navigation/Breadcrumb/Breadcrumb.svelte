<svelte:options runes={true} />

<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Breadcrumb.module.scss';

  type BreadcrumbItem = {
    label: string;
    href?: string;
    onClick?: () => void;
  };

  type Props = HTMLAttributes<HTMLElement> & {
    className?: string;
    items: BreadcrumbItem[];
    /** セパレーター文字列（デフォルト: '/'） */
    separator?: string;
  };

  let { items, className = undefined, separator = '/', ...restProps }: Props = $props();

  const rootClassName = $derived(joinClassName(styles.breadcrumb ?? '', className));

  function joinClassName(baseClassName: string, nextClassName?: string): string {
    return nextClassName === undefined || nextClassName === ''
      ? baseClassName
      : `${baseClassName} ${nextClassName}`;
  }

  function isLastItem(index: number): boolean {
    return index === items.length - 1;
  }
</script>

<nav class={rootClassName} aria-label="Breadcrumb" {...restProps}>
  <ol class={styles.list ?? ''}>
    {#each items as item, index (`${item.label}-${String(index)}`)}
      <li class={styles.item ?? ''}>
        {#if typeof item.href === 'string' && item.href !== ''}
          {#if isLastItem(index)}
            <a
              href={item.href}
              onclick={item.onClick}
              class={joinClassName(styles.link ?? '', styles.current ?? '')}
              aria-current="page"
            >
              {item.label}
            </a>
          {:else}
            <a href={item.href} onclick={item.onClick} class={styles.link ?? ''}>
              {item.label}
            </a>
          {/if}
        {:else}
          <span class={joinClassName(styles.link ?? '', isLastItem(index) ? (styles.current ?? '') : undefined)} aria-current={isLastItem(index) ? 'page' : undefined}>
            {item.label}
          </span>
        {/if}
        {#if !isLastItem(index)}
          <span class={styles.separator ?? ''} aria-hidden="true">{separator}</span>
        {/if}
      </li>
    {/each}
  </ol>
</nav>
