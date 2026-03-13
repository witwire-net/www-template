<script lang="ts">
  import { joinClassNames } from '@ui/components/navigation/shared';

  import styles from './Stepper.module.scss';

  type StepItem = {
    description?: string;
    label: string;
  };

  type Props = {
    activeStep?: number;
    steps: StepItem[];
  };

  let { steps, activeStep = 0 }: Props = $props();

  function hasDescription(description?: string): description is string {
    return description !== undefined && description !== '';
  }

  function getStatusClassName(index: number): string {
    return joinClassNames(
      styles.step ?? '',
      index < activeStep
        ? (styles.complete ?? '')
        : index === activeStep
          ? (styles.active ?? '')
          : (styles.pending ?? '')
    );
  }
</script>

<div class={styles.stepper ?? ''}>
  {#each steps as step, index (step.label)}
    <div class={getStatusClassName(index)}>
      <div class={styles.marker ?? ''}>{index + 1}</div>
      <div class={styles.text ?? ''}>
        <div class={styles.label ?? ''}>{step.label}</div>
        {#if hasDescription(step.description)}
          <div class={styles.description ?? ''}>{step.description}</div>
        {/if}
      </div>
      {#if index < steps.length - 1}
        <div class={styles.line ?? ''}></div>
      {/if}
    </div>
  {/each}
</div>
