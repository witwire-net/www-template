<script lang="ts">
  import Switch from '@ui/components/form/Switch/Switch.svelte';

  import styles from './PermissionEditor.module.scss';

  interface PermissionItem {
    id: string;
    label: string;
    description?: string;
    enabled: boolean;
  }

  interface Props {
    permissions?: readonly PermissionItem[];
    onToggle?: (id: string, value: boolean) => void;
  }

  let { permissions = [], onToggle = undefined }: Props = $props();

  const hasDescription = (description?: string): boolean => {
    return typeof description === 'string' && description !== '';
  };

  const handleToggle = (permissionId: string, event: Event): void => {
    const target = event.currentTarget;

    if (target instanceof HTMLInputElement) {
      onToggle?.(permissionId, target.checked);
    }
  };

  const createToggleHandler = (permissionId: string) => {
    return (event: Event): void => {
      handleToggle(permissionId, event);
    };
  };
</script>

<div class={styles.editor ?? ''}>
  {#each permissions as permission (permission.id)}
    <div class={styles.row ?? ''}>
      <div class={styles.content ?? ''}>
        <div class={styles.label ?? ''}>{permission.label}</div>
        {#if hasDescription(permission.description)}
          <div class={styles.description ?? ''}>{permission.description}</div>
        {/if}
      </div>
      <Switch checked={permission.enabled} onchange={createToggleHandler(permission.id)} />
    </div>
  {/each}
</div>
