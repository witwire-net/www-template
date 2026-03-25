<svelte:options runes={true} />

<script lang="ts">
  import { VirtualList } from '@ui/components/shared';

  import type { VirtualizeOptions } from '@ui/components/shared';

  import styles from './AuditLog.module.scss';

  type AuditLogEntry = {
    action: string;
    actor: string;
    target?: string;
    time: string;
  };

  type Props = {
    entries: AuditLogEntry[];
    /** When set, enables virtual scrolling for large lists. */
    virtualize?: VirtualizeOptions;
  };

  /** Matches `--spacing-xs` (0.25rem at 16px root). */
  const DEFAULT_GAP = 4;

  let { entries, virtualize }: Props = $props();

  function getEntryKey(entry: AuditLogEntry): string {
    const target = entry.target ?? '';

    return `${entry.actor}-${entry.time}-${target}`;
  }

  function getEntryKeyByIndex(index: number): string {
    return getEntryKey(getEntryByIndex(index));
  }

  function getEntryByIndex(index: number): AuditLogEntry {
    const entry = entries[index];

    if (entry === undefined) {
      throw new RangeError('Audit log entry index out of bounds');
    }

    return entry;
  }
</script>

{#snippet entryRow(entry: AuditLogEntry)}
  <div class={styles.entry ?? ''}>
    <div class={styles.actor ?? ''}>{entry.actor}</div>
    <div class={styles.action ?? ''}>{entry.action}</div>
    {#if entry.target !== undefined && entry.target !== ''}
      <div class={styles.target ?? ''}>{entry.target}</div>
    {/if}
    <div class={styles.time ?? ''}>{entry.time}</div>
  </div>
{/snippet}

{#if virtualize !== undefined}
  <VirtualList
    count={entries.length}
    estimateSize={virtualize.estimateSize}
    overscan={virtualize.overscan}
    height={virtualize.height}
    gap={virtualize.gap ?? DEFAULT_GAP}
    getItemKey={virtualize.getItemKey ?? getEntryKeyByIndex}
    className={styles.log ?? ''}
    ariaLabel="Audit log"
    row={(index) => entryRow(getEntryByIndex(index))}
  />
{:else}
  <div class={styles.log ?? ''}>
    {#each entries as entry (getEntryKey(entry))}
      {@render entryRow(entry)}
    {/each}
  </div>
{/if}
