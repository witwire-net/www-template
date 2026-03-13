<svelte:options runes={true} />

<script lang="ts">
  import Badge from '@ui/components/atoms/Badge/Badge.svelte';
  import Checkbox from '@ui/components/atoms/Checkbox/Checkbox.svelte';
  import Stack from '@ui/components/atoms/Stack/Stack.svelte';
  import Typography from '@ui/components/atoms/Typography/Typography.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';

  import styles from './TaskList.module.scss';

  type TaskItem = {
    completed?: boolean;
    due?: string;
    id: string;
    label: string;
  };

  type Props = {
    onToggle?: (id: string) => void;
    tasks: TaskItem[];
  };

  let { tasks, onToggle = undefined }: Props = $props();

  function getCheckboxId(taskId: string): string {
    return `task-checkbox-${taskId}`;
  }
</script>

<Stack direction="column" gap="sm">
  {#each tasks as task (task.id)}
    <Card className={styles.item ?? ''} padding="md">
      <div class={styles.labelWrapper ?? ''}>
        <Checkbox
          id={getCheckboxId(task.id)}
          checked={task.completed === true}
          onchange={() => onToggle?.(task.id)}
        />
        <label for={getCheckboxId(task.id)} class={styles.textContainer ?? ''}>
          <Typography
            as="span"
            variant="body"
            weight="medium"
            className={task.completed === true ? (styles.completed ?? '') : ''}
            color={task.completed === true ? 'muted' : 'default'}
          >
            {task.label}
          </Typography>
        </label>
      </div>
      {#if task.due !== undefined && task.due !== ''}
        <Badge
          variant={task.completed === true ? 'neutral' : 'primary'}
          size="sm"
          className={styles.due ?? ''}
        >
          {task.due}
        </Badge>
      {/if}
    </Card>
  {/each}
</Stack>
