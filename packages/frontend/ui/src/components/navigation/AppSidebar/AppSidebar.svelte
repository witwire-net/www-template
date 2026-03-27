<svelte:options runes={true} />

<script module lang="ts">
  let appSidebarSequence = 0;
</script>

<script lang="ts">
  import { IconX } from '@tabler/icons-svelte';

  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './AppSidebar.module.scss';

  type AppSidebarItem = {
    /** アクティブ状態 */
    active?: boolean;
    /** リンク先 URL（指定時は <a> として描画） */
    href?: string;
    /** クリックハンドラ（href がない場合は <button> として描画） */
    onClick?: () => void;
    /** 左側アイコン */
    icon?: Renderable;
    /** 表示ラベル */
    label: string;
    /** アイテム右端スロット（バッジ等は呼び出し側で実装） */
    trailing?: Renderable;
    /** サブアイテム（1段階のみ） */
    children?: AppSidebarItem[];
    /** サブメニューの初期展開状態（children がある場合のみ有効） */
    defaultExpanded?: boolean;
  };

  type Props = {
    className?: string;
    closeIcon?: Renderable;
    footer?: Renderable;
    /**
     * ナビリスト上部の見出しエリア（inline variant で SideNav の header に相当）
     * fixed variant ではロゴ表示に使用する
     */
    header?: Renderable;
    isOpen?: boolean;
    /** ナビゲーションアイテム一覧 */
    items?: AppSidebarItem[];
    onClose?: () => void;
    /**
     * 表示バリアント
     * - 'fixed'  : 画面左端に固定配置、モバイルでドロワー開閉（デフォルト）
     * - 'inline' : フロー内に配置、fixed/overlay/body-lock なし（旧 SideNav 相当）
     */
    variant?: 'fixed' | 'inline';
  };

  let {
    header = undefined,
    items = [],
    footer = undefined,
    className = undefined,
    isOpen = false,
    onClose = undefined,
    closeIcon = undefined,
    variant = 'fixed',
  }: Props = $props();

  const sidebarId = `app-sidebar-${String(++appSidebarSequence)}`;

  function shouldInitExpanded(item: AppSidebarItem): boolean {
    if (item.children === undefined || item.children.length === 0) {
      return false;
    }

    if (item.children.some((child) => child.active === true)) {
      return true;
    }

    return item.defaultExpanded === true;
  }

  let expandedMap = $state<Record<number, boolean>>({});

  $effect(() => {
    const map: Record<number, boolean> = {};

    for (const [index, item] of items.entries()) {
      map[index] = shouldInitExpanded(item);
    }

    expandedMap = map;
  });

  function toggleGroup(index: number): void {
    expandedMap = { ...expandedMap, [index]: !(expandedMap[index] ?? false) };
  }

  const isFixed = $derived(variant === 'fixed');
  const canClose = $derived(onClose !== undefined && isFixed);

  const sidebarClassName = $derived(
    joinClassNames(
      styles.sidebar ?? '',
      isFixed ? (styles.fixed ?? '') : (styles.inline ?? ''),
      isFixed && isOpen ? (styles.open ?? '') : undefined,
      className
    )
  );

  let isMobileViewport = $state(false);

  const overlayClassName = $derived(
    joinClassNames(styles.overlay ?? '', isOpen ? (styles.open ?? '') : undefined)
  );
  const isMobileSidebarHidden = $derived(isFixed && isMobileViewport && !isOpen);
  const showMobileOverlay = $derived(isFixed && isMobileViewport && isOpen);

  function handleClose(): void {
    onClose?.();
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (onClose === undefined) {
      return;
    }

    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Escape') {
      event.preventDefault();
      handleClose();
    }
  }

  $effect(() => {
    if (!isFixed || typeof window === 'undefined') {
      return;
    }

    const mediaQuery = window.matchMedia('(max-width: 768px)');

    const updateViewport = (): void => {
      isMobileViewport = mediaQuery.matches;
    };

    updateViewport();
    mediaQuery.addEventListener('change', updateViewport);

    return () => {
      mediaQuery.removeEventListener('change', updateViewport);
    };
  });

  $effect(() => {
    if (!isFixed || typeof document === 'undefined' || !isOpen) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  });

  function getItemClassName(item: AppSidebarItem): string {
    return joinClassNames(
      styles.navItem ?? '',
      item.active === true ? (styles.active ?? '') : undefined
    );
  }

  function getGroupPanelId(index: number): string {
    return `${sidebarId}-group-${String(index)}-panel`;
  }
</script>

{#snippet renderIconSlot(icon: Renderable)}
  {#if icon !== undefined && icon !== null}
    <span class={styles.navIcon ?? ''}>
      {#if isSnippet(icon)}
        {@render icon()}
      {:else}
        {getTextContent(icon)}
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet renderTrailing(trailing: Renderable)}
  {#if trailing !== undefined && trailing !== null}
    <span class={styles.trailing ?? ''}>
      {#if isSnippet(trailing)}
        {@render trailing()}
      {:else}
        {getTextContent(trailing)}
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet navList()}
  {#each items as item, index (item.label)}
    {#if item.children !== undefined && item.children.length > 0}
      <!-- グループアイテム：折りたたみ付き -->
      <div class={styles.group ?? ''}>
        <button
          type="button"
          class={joinClassNames(
            styles.navItem ?? '',
            styles.groupTrigger ?? '',
            item.active === true ? (styles.active ?? '') : undefined,
            (expandedMap[index] ?? false) ? (styles.groupTriggerOpen ?? '') : undefined
          )}
          aria-expanded={expandedMap[index] ?? false}
          aria-controls={getGroupPanelId(index)}
          onclick={() => {
            toggleGroup(index);
          }}
        >
          {@render renderIconSlot(item.icon)}
          <span class={styles.groupLabel ?? ''}>{item.label}</span>
          {@render renderTrailing(item.trailing)}
          <span
            class={joinClassNames(
              styles.chevron ?? '',
              (expandedMap[index] ?? false) ? (styles.chevronOpen ?? '') : undefined
            )}
            aria-hidden="true"
          ></span>
        </button>

        <div
          id={getGroupPanelId(index)}
          role="region"
          class={joinClassNames(
            styles.groupPanel ?? '',
            (expandedMap[index] ?? false) ? (styles.groupPanelOpen ?? '') : undefined
          )}
        >
          <div class={styles.groupPanelInner ?? ''}>
            {#each item.children as child (child.label)}
              {#if child.href !== undefined && child.href !== ''}
                <a
                  href={child.href}
                  onclick={child.onClick}
                  class={joinClassNames(getItemClassName(child), styles.childItem ?? '')}
                >
                  {@render renderIconSlot(child.icon)}
                  <span class={styles.childLabel ?? ''}>{child.label}</span>
                  {@render renderTrailing(child.trailing)}
                </a>
              {:else if child.onClick !== undefined}
                <button
                  type="button"
                  onclick={child.onClick}
                  class={joinClassNames(getItemClassName(child), styles.childItem ?? '', styles.itemButton ?? '')}
                >
                  {@render renderIconSlot(child.icon)}
                  <span class={styles.childLabel ?? ''}>{child.label}</span>
                  {@render renderTrailing(child.trailing)}
                </button>
              {:else}
                <span class={joinClassNames(getItemClassName(child), styles.childItem ?? '', styles.itemStatic ?? '')}>
                  {@render renderIconSlot(child.icon)}
                  <span class={styles.childLabel ?? ''}>{child.label}</span>
                  {@render renderTrailing(child.trailing)}
                </span>
              {/if}
            {/each}
          </div>
        </div>
      </div>
    {:else if item.href !== undefined && item.href !== ''}
      <!-- リンクアイテム -->
      <a href={item.href} onclick={item.onClick} class={getItemClassName(item)}>
        {@render renderIconSlot(item.icon)}
        <span class={styles.itemLabel ?? ''}>{item.label}</span>
        {@render renderTrailing(item.trailing)}
      </a>
    {:else if item.onClick !== undefined}
      <!-- ボタンアイテム -->
      <button
        type="button"
        onclick={item.onClick}
        class={joinClassNames(getItemClassName(item), styles.itemButton ?? '')}
      >
        {@render renderIconSlot(item.icon)}
        <span class={styles.itemLabel ?? ''}>{item.label}</span>
        {@render renderTrailing(item.trailing)}
      </button>
    {:else}
      <!-- 静的アイテム -->
      <span class={joinClassNames(getItemClassName(item), styles.itemStatic ?? '')}>
        {@render renderIconSlot(item.icon)}
        <span class={styles.itemLabel ?? ''}>{item.label}</span>
        {@render renderTrailing(item.trailing)}
      </span>
    {/if}
  {/each}
{/snippet}

{#if showMobileOverlay && canClose}
  <button
    type="button"
    class={overlayClassName}
    aria-label="Close sidebar"
    onclick={handleClose}
    onkeydown={handleBackdropKeydown}
  ></button>
{/if}

<aside
  class={sidebarClassName}
  inert={isMobileSidebarHidden ? true : undefined}
  aria-hidden={isMobileSidebarHidden ? 'true' : undefined}
>
  <!-- ヘッダーエリア：fixed ではロゴ＋閉じるボタン、inline では見出し -->
  {#if header !== undefined && header !== null || canClose}
    <div class={styles.sidebarHeader ?? ''}>
      {#if header !== undefined && header !== null}
        <div class={isFixed ? (styles.logo ?? '') : (styles.inlineHeader ?? '')}>
          {#if isSnippet(header)}
            {@render header()}
          {:else}
            {getTextContent(header)}
          {/if}
        </div>
      {/if}
      {#if canClose}
        <button
          type="button"
          class={styles.closeParams ?? ''}
          onclick={handleClose}
          aria-label="Close sidebar"
        >
          {#if closeIcon !== undefined && closeIcon !== null}
            {#if isSnippet(closeIcon)}
              {@render closeIcon()}
            {:else}
              {getTextContent(closeIcon)}
            {/if}
          {:else}
            <Icon icon={IconX} className={styles.closeIcon ?? ''} title="Close sidebar" />
          {/if}
        </button>
      {/if}
    </div>
  {/if}

  <nav class={styles.nav ?? ''}>
    {@render navList()}
  </nav>

  {#if footer !== undefined && footer !== null}
    <div class={styles.footer ?? ''}>
      {#if isSnippet(footer)}
        {@render footer()}
      {:else}
        {getTextContent(footer)}
      {/if}
    </div>
  {/if}
</aside>
