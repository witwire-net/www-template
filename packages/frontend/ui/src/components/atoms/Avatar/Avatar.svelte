<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Avatar.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    alt?: string;
    className?: string;
    name?: string;
    shape?: 'circle' | 'rounded';
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
    src?: string;
    status?: 'online' | 'offline' | 'busy' | 'away';
  };

  let {
    src = undefined,
    alt = undefined,
    name = undefined,
    size = 'md',
    status = undefined,
    shape = 'circle',
    className = undefined,
    ...restProps
  }: Props = $props();

  function getInitials(value?: string): string {
    if (value === undefined || value === '') {
      return '';
    }

    return value
      .trim()
      .split(' ')
      .slice(0, 2)
      .map((part) => part[0] ?? '')
      .join('')
      .toUpperCase();
  }

  const rootClassName = $derived(
    [styles.avatar ?? '', styles[size] ?? '', styles[shape] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const hasImage = $derived(src !== undefined && src !== '');
  const hasStatus = $derived(status !== undefined);
  const resolvedAlt = $derived(
    alt !== undefined && alt !== '' ? alt : name !== undefined && name !== '' ? name : 'Avatar'
  );
  const initials = $derived(getInitials(name));
</script>

<div class={rootClassName} {...restProps}>
  {#if hasImage}
    <img src={src} alt={resolvedAlt} class={styles.image ?? ''} />
  {:else}
    <span class={styles.initials ?? ''}>{initials}</span>
  {/if}
  {#if hasStatus}
    <span class={`${styles.status ?? ''} ${status === undefined ? '' : (styles[status] ?? '')}`}></span>
  {/if}
</div>
