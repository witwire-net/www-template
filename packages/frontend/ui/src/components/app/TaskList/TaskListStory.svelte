<svelte:options runes={true} />

<script lang="ts">
  import TaskList from './TaskList.svelte';

  type StoryTask = {
    completed?: boolean;
    due?: string;
    id: string;
    label: string;
  };

  let tasks = $state<StoryTask[]>([
    { id: '1', label: 'Review Q1 report', completed: true },
    { id: '2', label: 'Invite finance team' },
    { id: '3', label: 'Update billing email', due: 'Due tomorrow' },
  ]);

  function handleToggle(id: string): void {
    tasks = tasks.map((task) =>
      task.id === id ? { ...task, completed: !(task.completed === true) } : task
    );
  }
</script>

<TaskList {tasks} onToggle={handleToggle} />
