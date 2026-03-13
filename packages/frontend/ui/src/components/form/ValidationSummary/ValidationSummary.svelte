<svelte:options runes={true} />

<script lang="ts">
  import styles from './ValidationSummary.module.scss';

  type Props = Record<string, unknown> & {
    title?: string;
    errors?: string[];
    className?: string;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    title = 'Please fix the following',
    errors = [],
    className = undefined,
    ...restProps
  }: Props = $props();

  const hasErrors = $derived(errors.length > 0);
  const summaryClassName = $derived(joinClassName(styles.summary, className));
</script>

{#if hasErrors}
  <div class={summaryClassName} role="alert" {...restProps}>
    <div class={styles.title ?? ''}>{title}</div>
    <ul class={styles.list ?? ''}>
      {#each errors as error, index (`${error}-${index}`)}
        <li class={styles.item ?? ''}>{error}</li>
      {/each}
    </ul>
  </div>
{/if}
