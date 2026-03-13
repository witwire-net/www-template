<script lang="ts">
  import DataGrid from './DataGrid.svelte';

  type UserRow = {
    id: number;
    name: string;
    role: string;
    status: string;
  };

  const rows: readonly UserRow[] = [
    { id: 1, name: 'Jordan Lee', role: 'Admin', status: 'Active' },
    { id: 2, name: 'Sam Park', role: 'Editor', status: 'Invited' },
    { id: 3, name: 'Tara Singh', role: 'Viewer', status: 'Active' },
  ];

  const columns = [
    { header: 'Name', accessor: nameCell },
    { header: 'Role', accessor: roleCell },
    { header: 'Status', accessor: statusCell },
  ] as const;

  let { compact = false }: { compact?: boolean } = $props();
</script>

<div style="width: min(100%, 960px);">
  <DataGrid columns={columns} data={rows} getRowKey={(row) => row.id} {compact}></DataGrid>
</div>

{#snippet nameCell(row: UserRow)}
  <strong>{row.name}</strong>
{/snippet}

{#snippet roleCell(row: UserRow)}
  <span>{row.role}</span>
{/snippet}

{#snippet statusCell(row: UserRow)}
  <span>{row.status}</span>
{/snippet}
