<svelte:options runes={true} />

<script module lang="ts">
  let tabSequence = 0;
</script>

<script lang="ts">
  import { joinClassNames } from '@ui/components/navigation/shared';

  import styles from './Tabs.module.scss';

  type TabItem = {
    disabled?: boolean;
    label: string;
    value: string;
  };

  type Props = {
    /** aria-label for the tablist */
    ariaLabel?: string;
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
    ariaLabel = undefined,
  }: Props = $props();

  const tabId = `tabs-${String(++tabSequence)}`;

  let internalValue = $state('');
  let isInitialized = $state(false);

  const activeValue = $derived(value ?? internalValue);
  const rootClassName = $derived(joinClassNames(styles.tabs ?? '', className));

  function getTabId(tabValue: string): string {
    return `${tabId}-tab-${tabValue}`;
  }

  function getPanelId(tabValue: string): string {
    return `${tabId}-panel-${tabValue}`;
  }

  function handleSelect(next: string, disabled?: boolean): void {
    if (disabled === true) {
      return;
    }

    if (value === undefined) {
      internalValue = next;
    }

    onChange?.(next);
  }

  function handleKeydown(event: KeyboardEvent, currentValue: string): void {
    const enabledItems = items.filter((item) => item.disabled !== true);
    const currentIndex = enabledItems.findIndex((item) => item.value === currentValue);

    let nextIndex: number | undefined;

    if (event.key === 'ArrowRight') {
      event.preventDefault();
      nextIndex = (currentIndex + 1) % enabledItems.length;
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault();
      nextIndex = (currentIndex - 1 + enabledItems.length) % enabledItems.length;
    } else if (event.key === 'Home') {
      event.preventDefault();
      nextIndex = 0;
    } else if (event.key === 'End') {
      event.preventDefault();
      nextIndex = enabledItems.length - 1;
    }

    if (nextIndex !== undefined) {
      const nextItem = enabledItems[nextIndex];

      if (nextItem !== undefined) {
        handleSelect(nextItem.value);

        // フォーカスを次のタブに移動
        const nextTabEl = document.getElementById(getTabId(nextItem.value));
        nextTabEl?.focus();
      }
    }
  }

  $effect(() => {
    if (isInitialized) {
      return;
    }

    internalValue = defaultValue ?? items[0]?.value ?? '';
    isInitialized = true;
  });
</script>

<div class={rootClassName} role="tablist" aria-label={ariaLabel}>
  {#each items as item (item.value)}
    <button
      id={getTabId(item.value)}
      type="button"
      role="tab"
      class={joinClassNames(
        styles.tab ?? '',
        activeValue === item.value ? (styles.active ?? '') : undefined,
        item.disabled === true ? (styles.disabled ?? '') : undefined
      )}
      aria-selected={activeValue === item.value}
      aria-controls={getPanelId(item.value)}
      tabindex={activeValue === item.value ? 0 : -1}
      onclick={() => {
        handleSelect(item.value, item.disabled);
      }}
      onkeydown={(event) => {
        handleKeydown(event, item.value);
      }}
      disabled={item.disabled === true}
    >
      {item.label}
    </button>
  {/each}
</div>
