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
  };

  let { items, className = undefined, ...restProps }: Props = $props();

  const rootClassName = $derived(joinClassName(styles.breadcrumb ?? '', className));

  function joinClassName(baseClassName: string, nextClassName?: string): string {
    return nextClassName === undefined || nextClassName === ''
      ? baseClassName
      : `${baseClassName} ${nextClassName}`;
  }
</script>

<nav class={rootClassName} {...restProps}>
  {#each items as item, index (`${item.label}-${String(index)}`)}
    <span class={styles.item ?? ''}>
      {#if typeof item.href === 'string' && item.href !== ''}
        <a href={item.href} onclick={item.onClick} class={styles.link ?? ''}>
          {item.label}
        </a>
      {:else}
        <span class={styles.current ?? ''}>{item.label}</span>
      {/if}
      {#if index < items.length - 1}
        <span class={styles.separator ?? ''}>/</span>
      {/if}
    </span>
  {/each}
</nav>
