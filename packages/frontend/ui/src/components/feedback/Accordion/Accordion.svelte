<svelte:options runes={true} />

<script module lang="ts">
  let accordionSequence = 0;
</script>

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Accordion.module.scss';

  type Renderable = Snippet | string | number | null | undefined;

  type AccordionItem = {
    title: Renderable;
    content: Renderable;
    id?: string;
  };

  type Props = HTMLAttributes<HTMLDivElement> & {
    items: AccordionItem[];
    defaultOpenIndexes?: number[];
    allowMultipleOpen?: boolean;
    className?: string;
    itemClassName?: string;
    openItemClassName?: string;
    triggerClassName?: string;
    titleClassName?: string;
    indicatorClassName?: string;
    indicatorOpenClassName?: string;
    panelClassName?: string;
    panelOpenClassName?: string;
    contentClassName?: string;
  };

  const EMPTY_INDEXES: number[] = [];

  let {
    items,
    defaultOpenIndexes = EMPTY_INDEXES,
    allowMultipleOpen = true,
    className = undefined,
    itemClassName = undefined,
    openItemClassName = undefined,
    triggerClassName = undefined,
    titleClassName = undefined,
    indicatorClassName = undefined,
    indicatorOpenClassName = undefined,
    panelClassName = undefined,
    panelOpenClassName = undefined,
    contentClassName = undefined,
    ...restProps
  }: Props = $props();

  const accordionId = `accordion-${String(++accordionSequence)}`;

  let openIndexes = $state<number[]>([]);

  function normalizeIndexes(indexes: number[], maxIndex: number): number[] {
    return Array.from(new Set(indexes)).filter(
      (index) => Number.isInteger(index) && index >= 0 && index <= maxIndex
    );
  }

  function isSnippet(value: Renderable): value is Snippet {
    return typeof value === 'function';
  }

  function getTextContent(value: Renderable): string {
    if (typeof value === 'string') {
      return value;
    }

    if (typeof value === 'number') {
      return String(value);
    }

    return '';
  }

  function toggle(index: number): void {
    const isOpen = openIndexes.includes(index);

    if (allowMultipleOpen) {
      openIndexes = isOpen
        ? openIndexes.filter((openIndex) => openIndex !== index)
        : [...openIndexes, index];
      return;
    }

    openIndexes = isOpen ? [] : [index];
  }

  const normalizedDefaultOpenIndexes = $derived(normalizeIndexes(defaultOpenIndexes, items.length - 1));

  const rootClassName = $derived(
    [styles.accordion ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );

  $effect(() => {
    openIndexes = normalizedDefaultOpenIndexes;
  });
</script>

<div class={rootClassName} {...restProps}>
  {#each items as item, index (item.id ?? String(index))}
    <div
      class={[
        styles.item ?? '',
        itemClassName ?? '',
        openIndexes.includes(index) ? (styles.itemOpen ?? '') : '',
        openIndexes.includes(index) ? (openItemClassName ?? '') : '',
      ]
        .filter((value) => value !== '')
        .join(' ')}
    >
      <button
        type="button"
        class={[styles.trigger ?? '', triggerClassName ?? ''].filter((value) => value !== '').join(' ')}
        id={`${accordionId}-trigger-${String(index)}`}
        aria-expanded={openIndexes.includes(index)}
        aria-controls={`${accordionId}-panel-${String(index)}`}
        onclick={() => {
          toggle(index);
        }}
      >
        <span class={[styles.title ?? '', titleClassName ?? ''].filter((value) => value !== '').join(' ')}>
          {#if isSnippet(item.title)}
            {@render item.title()}
          {:else}
            {getTextContent(item.title)}
          {/if}
        </span>
        <span
          class={[
            styles.indicator ?? '',
            indicatorClassName ?? '',
            openIndexes.includes(index) ? (styles.indicatorOpen ?? '') : '',
            openIndexes.includes(index) ? (indicatorOpenClassName ?? '') : '',
          ]
            .filter((value) => value !== '')
            .join(' ')}
          aria-hidden="true"
        ></span>
      </button>
      <div
        id={`${accordionId}-panel-${String(index)}`}
        role="region"
        aria-labelledby={`${accordionId}-trigger-${String(index)}`}
        class={[
          styles.panel ?? '',
          panelClassName ?? '',
          openIndexes.includes(index) ? (styles.panelOpen ?? '') : '',
          openIndexes.includes(index) ? (panelOpenClassName ?? '') : '',
        ]
          .filter((value) => value !== '')
          .join(' ')}
      >
        <div class={[styles.content ?? '', contentClassName ?? ''].filter((value) => value !== '').join(' ')}>
          {#if isSnippet(item.content)}
            {@render item.content()}
          {:else}
            {getTextContent(item.content)}
          {/if}
        </div>
      </div>
    </div>
  {/each}
</div>
