<svelte:options runes={true} />

<script lang="ts">
  import { joinClassNames } from '@ui/components/navigation/shared';

  import styles from './Tabs.module.scss';

  type TabItem = {
    disabled?: boolean;
    label: string;
    value: string;
  };

  type Props = {
    className?: string;
    defaultValue?: string;
    items: TabItem[];
    onChange?: (value: string) => void;
    value?: string;
  };

  let {
    items,
    value = undefined,
    defaultValue = undefined,
    onChange = undefined,
    className = undefined,
  }: Props = $props();

  let internalValue = $state('');
  let isInitialized = $state(false);

  const activeValue = $derived(value ?? internalValue);
  const rootClassName = $derived(joinClassNames(styles.tabs ?? '', className));

  function handleSelect(next: string, disabled?: boolean): void {
    if (disabled === true) {
      return;
    }

    if (value === undefined) {
      internalValue = next;
    }

    onChange?.(next);
  }

  $effect(() => {
    if (isInitialized) {
      return;
    }

    internalValue = defaultValue ?? items[0]?.value ?? '';
    isInitialized = true;
  });
</script>

<div class={rootClassName} role="tablist">
  {#each items as item (item.value)}
    <button
      type="button"
      role="tab"
      class={joinClassNames(
        styles.tab ?? '',
        activeValue === item.value ? (styles.active ?? '') : undefined,
        item.disabled === true ? (styles.disabled ?? '') : undefined
      )}
      aria-selected={activeValue === item.value}
      onclick={() => {
        handleSelect(item.value, item.disabled);
      }}
      disabled={item.disabled === true}
    >
      {item.label}
    </button>
  {/each}
</div>
