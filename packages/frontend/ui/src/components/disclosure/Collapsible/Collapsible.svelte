<svelte:options runes={true} />

<script module lang="ts">
  let collapsibleSequence = 0;
</script>

<script lang="ts">
  import { untrack } from 'svelte';

  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/shared';

  import styles from './Collapsible.module.scss';

  type Props = {
    /** トリガー要素（ボタンとして描画）に表示するコンテンツ */
    trigger: Renderable;
    /** 展開時に表示するコンテンツ */
    content: Renderable;
    /** 制御モード：外部から展開状態を渡す */
    open?: boolean;
    /** 非制御モード：初期展開状態 */
    defaultOpen?: boolean;
    /** 展開状態が変わったときのコールバック */
    onOpenChange?: (open: boolean) => void;
    /** ルート要素への追加クラス */
    className?: string;
    /** トリガーボタンへの追加クラス */
    triggerClassName?: string;
    /** コンテンツパネルへの追加クラス */
    contentClassName?: string;
    /** トリガーの aria-label（trigger が文字列でない場合に設定推奨） */
    triggerAriaLabel?: string;
  };

  let {
    trigger,
    content,
    open = undefined,
    defaultOpen = false,
    onOpenChange = undefined,
    className = undefined,
    triggerClassName = undefined,
    contentClassName = undefined,
    triggerAriaLabel = undefined,
  }: Props = $props();

  const collapsibleId = `collapsible-${String(++collapsibleSequence)}`;
  const panelId = `${collapsibleId}-panel`;

  let internalOpen = $state(untrack(() => defaultOpen));
  const isOpen = $derived(open ?? internalOpen);

  const rootClassName = $derived(joinClassNames(styles.collapsible ?? '', className));
  const triggerClass = $derived(
    joinClassNames(styles.trigger ?? '', isOpen ? (styles.triggerOpen ?? '') : undefined, triggerClassName)
  );
  const panelClass = $derived(
    joinClassNames(styles.panel ?? '', isOpen ? (styles.panelOpen ?? '') : undefined, contentClassName)
  );

  function toggle(): void {
    const next = !isOpen;

    if (open === undefined) {
      internalOpen = next;
    }

    onOpenChange?.(next);
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      toggle();
    }
  }
</script>

<div class={rootClassName}>
  <button
    type="button"
    class={triggerClass}
    aria-expanded={isOpen}
    aria-controls={panelId}
    aria-label={triggerAriaLabel}
    onclick={toggle}
    onkeydown={handleKeydown}
  >
    <span class={styles.triggerContent ?? ''}>
      {#if isSnippet(trigger)}
        {@render trigger()}
      {:else}
        {getTextContent(trigger)}
      {/if}
    </span>
    <span class={joinClassNames(styles.indicator ?? '', isOpen ? (styles.indicatorOpen ?? '') : undefined)} aria-hidden="true"></span>
  </button>
  <div
    id={panelId}
    role="region"
    class={panelClass}
  >
    <div class={styles.panelInner ?? ''}>
      {#if isSnippet(content)}
        {@render content()}
      {:else}
        {getTextContent(content)}
      {/if}
    </div>
  </div>
</div>
