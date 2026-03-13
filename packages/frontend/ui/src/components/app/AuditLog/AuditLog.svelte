<svelte:options runes={true} />

<script lang="ts">
  import styles from './AuditLog.module.scss';

  type AuditLogEntry = {
    action: string;
    actor: string;
    target?: string;
    time: string;
  };

  type Props = {
    entries: AuditLogEntry[];
  };

  let { entries }: Props = $props();

  function getEntryKey(entry: AuditLogEntry): string {
    const target = entry.target ?? '';

    return `${entry.actor}-${entry.time}-${target}`;
  }
</script>

<div class={styles.log ?? ''}>
  {#each entries as entry (getEntryKey(entry))}
    <div class={styles.entry ?? ''}>
      <div class={styles.actor ?? ''}>{entry.actor}</div>
      <div class={styles.action ?? ''}>{entry.action}</div>
      {#if entry.target !== undefined && entry.target !== ''}
        <div class={styles.target ?? ''}>{entry.target}</div>
      {/if}
      <div class={styles.time ?? ''}>{entry.time}</div>
    </div>
  {/each}
</div>
