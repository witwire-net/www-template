<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './APIKeyManager.module.scss';

  type ApiKeyRenderable = Snippet | string | number | null | undefined;

  interface ApiKeyField {
    label?: string;
    value: string;
  }

  interface ApiKeyItem {
    id: string | number;
    name?: string;
    maskedValue?: string;
    description?: string;
    date?: ApiKeyField;
    metadata?: readonly ApiKeyField[];
    actions?: readonly ApiKeyRenderable[];
    label?: string;
    maskedKey?: string;
    createdAt?: string;
    action?: ApiKeyRenderable;
  }

  interface APIKeyManagerProps extends HTMLAttributes<HTMLDivElement> {
    keys?: readonly ApiKeyItem[];
    className?: string;
  }

  let { keys = [], className, ...restProps }: APIKeyManagerProps = $props();

  const rootClassName = $derived([styles.list ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));

  const hasText = (value: string | undefined): value is string => value !== undefined && value !== '';

  const hasValue = (value: ApiKeyRenderable): boolean => value !== undefined && value !== null && value !== '';

  const isSnippet = (value: ApiKeyRenderable): value is Snippet =>
    typeof value === 'function';

  const renderValue = (value: ApiKeyRenderable): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const resolveName = (key: ApiKeyItem): string => {
    return key.name ?? key.label ?? '';
  };

  const resolveMaskedValue = (key: ApiKeyItem): string => {
    return key.maskedValue ?? key.maskedKey ?? '';
  };

  const resolveDate = (key: ApiKeyItem): ApiKeyField | undefined => {
    if (key.date !== undefined) {
      return key.date;
    }

    const createdAt = key.createdAt;

    if (hasText(createdAt)) {
      return {
        label: 'Created',
        value: createdAt,
      };
    }

    return undefined;
  };

  const resolveActions = (key: ApiKeyItem): readonly ApiKeyRenderable[] => {
    if (key.actions !== undefined) {
      return key.actions;
    }

    if (key.action !== undefined) {
      return [key.action];
    }

    return [];
  };
</script>

<div class={rootClassName} {...restProps}>
  {#each keys as key (key.id)}
    {@const date = resolveDate(key)}
    {@const actions = resolveActions(key)}
    <div class={styles.row}>
      <div class={styles.summary}>
        <div class={styles.label}>{resolveName(key)}</div>
        {#if hasText(key.description)}
          <div class={styles.description}>{key.description}</div>
        {/if}
        {#if key.metadata !== undefined && key.metadata.length > 0}
          <div class={styles.metadata}>
            {#each key.metadata as item, index (`${key.id}-meta-${item.label ?? item.value}-${String(index)}`)}
              <span class={styles.metaItem}>
                {#if hasText(item.label)}
                  <span class={styles.metaLabel}>{item.label}:</span>
                {/if}
                <span>{item.value}</span>
              </span>
            {/each}
          </div>
        {/if}
      </div>
      <div class={styles.key}>{resolveMaskedValue(key)}</div>
      {#if date !== undefined}
        <div class={styles.date}>
          {#if hasText(date.label)}
            <span class={styles.metaLabel}>{date.label}:</span>
          {/if}
          <span>{date.value}</span>
        </div>
      {/if}
      {#if actions.length > 0}
        <div class={styles.actions}>
          {#each actions as action, index (`${key.id}-action-${String(index)}`)}
            {#if hasValue(action)}
              <div class={styles.action}>
                {#if isSnippet(action)}
                  {@render action()}
                {:else}
                  {renderValue(action)}
                {/if}
              </div>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>
