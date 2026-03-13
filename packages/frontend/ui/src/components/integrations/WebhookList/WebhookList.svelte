<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import Badge from '@ui/components/atoms/Badge/Badge.svelte';

  import styles from './WebhookList.module.scss';

  type WebhookRenderable = Snippet | string | number | null | undefined;
  type WebhookTone = 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';

  interface WebhookStatus {
    label: string;
    tone?: WebhookTone;
  }

  interface WebhookMetadataItem {
    label?: string;
    value: string;
  }

  interface WebhookItem {
    id: string | number;
    endpoint?: string;
    title?: string;
    summary?: string;
    status?: 'active' | 'paused' | string | WebhookStatus;
    metadata?: readonly WebhookMetadataItem[];
    actions?: readonly WebhookRenderable[];
    url?: string;
    events?: string;
  }

  interface WebhookListProps extends HTMLAttributes<HTMLDivElement> {
    webhooks?: readonly WebhookItem[];
    className?: string;
  }

  let { webhooks = [], className, ...restProps }: WebhookListProps = $props();

  const rootClassName = $derived([styles.list ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));

  const hasText = (value: string | undefined): value is string => value !== undefined && value !== '';

  const hasRenderable = (value: WebhookRenderable): boolean => {
    return value !== undefined && value !== null && value !== '';
  };

  const isSnippet = (value: WebhookRenderable): value is Snippet => {
    return typeof value === 'function';
  };

  const renderValue = (value: WebhookRenderable): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return String(value);
    }

    return '';
  };

  const resolveStatus = (hook: WebhookItem): WebhookStatus => {
    if (typeof hook.status === 'string') {
      if (hook.status === 'active') {
        return { label: 'Active', tone: 'success' };
      }

      if (hook.status === 'paused') {
        return { label: 'Paused', tone: 'warning' };
      }

      return { label: hook.status, tone: 'neutral' };
    }

    if (hook.status !== undefined) {
      return {
        label: hook.status.label,
        tone: hook.status.tone ?? 'neutral',
      };
    }

    return { label: 'Unknown', tone: 'neutral' };
  };

  const resolveMetadata = (hook: WebhookItem): readonly WebhookMetadataItem[] => {
    if (hook.metadata !== undefined && hook.metadata.length > 0) {
      return hook.metadata;
    }

    const events = hook.events;

    if (hasText(events)) {
      return [{ label: 'Events', value: events }];
    }

    return [];
  };

  const resolveEndpoint = (hook: WebhookItem): string => {
    return hook.endpoint ?? hook.url ?? '';
  };
</script>

<div class={rootClassName} {...restProps}>
  {#each webhooks as hook (hook.id)}
    {@const status = resolveStatus(hook)}
    {@const metadata = resolveMetadata(hook)}
    <div class={styles.row}>
      <div class={styles.summary}>
        {#if hasText(hook.title)}
          <div class={styles.title}>{hook.title}</div>
        {/if}
        <div class={styles.url}>{resolveEndpoint(hook)}</div>
        {#if hasText(hook.summary)}
          <div class={styles.description}>{hook.summary}</div>
        {/if}
        {#if metadata.length > 0}
          <div class={styles.metadata}>
            {#each metadata as item, index (`${hook.id}-meta-${item.label ?? item.value}-${String(index)}`)}
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
      <div class={styles.status}>
        <Badge variant={status.tone} size="sm">{status.label}</Badge>
      </div>
      {#if hook.actions !== undefined && hook.actions.length > 0}
        <div class={styles.actions}>
          {#each hook.actions as action, index (`${hook.id}-action-${String(index)}`)}
            {#if hasRenderable(action)}
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
