<svelte:options runes={true} />

<script lang="ts">
  import styles from './ProgressOverlay.module.scss';

  type Props = {
    open: boolean;
    message?: string;
  };

  let { open, message = 'Processing' }: Props = $props();
</script>

{#if open}
  <div class={styles.overlay ?? ''} role="alert" aria-busy="true">
    <div class={styles.card ?? ''}>
      <span class={styles.spinner ?? ''} aria-hidden="true"></span>
      <span class={styles.message ?? ''}>{message}</span>
    </div>
  </div>
{/if}
