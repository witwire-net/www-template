<svelte:options runes={true} />

<script lang="ts">
  import PermissionEditor from '@ui/components/admin/PermissionEditor/PermissionEditor.svelte';

  let permissions = $state([
    {
      id: 'create',
      label: 'Create projects',
      enabled: true,
      description: 'Allow creating new projects',
    },
    {
      id: 'delete',
      label: 'Delete projects',
      enabled: false,
      description: 'Allow deleting projects',
    },
    {
      id: 'billing',
      label: 'Manage billing',
      enabled: true,
    },
  ]);

  const handleToggle = (id: string, value: boolean): void => {
    permissions = permissions.map((permission) => {
      if (permission.id === id) {
        return { ...permission, enabled: value };
      }

      return permission;
    });
  };
</script>

<PermissionEditor {permissions} onToggle={handleToggle} />
