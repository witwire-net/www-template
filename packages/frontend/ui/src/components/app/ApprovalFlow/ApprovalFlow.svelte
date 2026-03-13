<svelte:options runes={true} />

<script lang="ts">
  import { joinClassNames } from '@ui/components/app/shared';

  import styles from './ApprovalFlow.module.scss';

  type ApprovalStep = {
    approver?: string;
    label: string;
    status: 'pending' | 'approved' | 'rejected';
  };

  type Props = {
    steps: ApprovalStep[];
  };

  let { steps }: Props = $props();

  function getStepClassName(status: ApprovalStep['status']): string {
    return joinClassNames(styles.step ?? '', styles[status] ?? '');
  }

  function getStepKey(step: ApprovalStep): string {
    return `${step.label}-${step.status}`;
  }
</script>

<div class={styles.flow ?? ''}>
  {#each steps as step (getStepKey(step))}
    <div class={getStepClassName(step.status)}>
      <div class={styles.marker ?? ''}></div>
      <div class={styles.content ?? ''}>
        <div class={styles.label ?? ''}>{step.label}</div>
        {#if step.approver !== undefined && step.approver !== ''}
          <div class={styles.approver ?? ''}>{step.approver}</div>
        {/if}
      </div>
    </div>
  {/each}
</div>
