<svelte:options runes={true} />

<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  import { joinClassNames } from '@ui/components/app/shared';

  import styles from './SearchBar.module.scss';

  type Props = Omit<HTMLInputAttributes, 'type'> & {
    className?: string;
    onSearch?: (value: string) => void;
  };

  let {
    onSearch = undefined,
    className = undefined,
    placeholder = 'Search',
    oninput = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(joinClassNames(styles.search ?? '', className));

  function handleInput(event: Event & { currentTarget: EventTarget & HTMLInputElement }): void {
    onSearch?.(event.currentTarget.value);
    oninput?.(event);
  }
</script>

<div class={rootClassName}>
  <input
    {...restProps}
    type="search"
    class={styles.input ?? ''}
    placeholder={placeholder}
    oninput={handleInput}
  />
</div>
