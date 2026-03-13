<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './Popover.module.scss';

  type Renderable = Snippet | string | number | null | undefined;

  type Props = {
    trigger?: Renderable;
    content?: Renderable;
    placement?: 'top' | 'bottom' | 'left' | 'right';
  };

  let { trigger = undefined, content = undefined, placement = 'bottom' }: Props = $props();

  let open = $state(false);
  let wrapperElement = $state<HTMLDivElement | null>(null);

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

  function toggle(): void {
    open = !open;
  }

  function handleTriggerKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggle();
    }
  }

  const placementClassName = $derived(styles[placement] ?? styles.bottom ?? '');

  $effect(() => {
    if (typeof document === 'undefined' || !open || wrapperElement === null) {
      return;
    }

    const handleMousedown = (event: MouseEvent): void => {
      if (!(event.target instanceof Node)) {
        return;
      }

      if (!wrapperElement?.contains(event.target)) {
        open = false;
      }
    };

    document.addEventListener('mousedown', handleMousedown);

    return () => {
      document.removeEventListener('mousedown', handleMousedown);
    };
  });
</script>

<div class={styles.wrapper ?? ''} bind:this={wrapperElement}>
  <div
    class={styles.trigger ?? ''}
    role="button"
    tabindex="0"
    onclick={toggle}
    onkeydown={handleTriggerKeydown}
  >
    {#if isSnippet(trigger)}
      {@render trigger()}
    {:else}
      {getTextContent(trigger)}
    {/if}
  </div>
  {#if open}
    <div class={[styles.popover ?? '', placementClassName].filter((value) => value !== '').join(' ')}>
      {#if isSnippet(content)}
        {@render content()}
      {:else}
        {getTextContent(content)}
      {/if}
    </div>
  {/if}
</div>
