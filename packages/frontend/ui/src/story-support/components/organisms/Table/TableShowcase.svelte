<script lang="ts">
  import Table from '@ui/components/organisms/Table/Table.svelte';

  type TableRow = {
    id: string;
    user: string;
    task: string;
    date: string;
    status: 'Completed' | 'Pending' | 'Review';
  };

  const rows: readonly TableRow[] = [
    {
      id: 'row-1',
      user: 'Aki Tanaka',
      task: 'Created subscription report',
      date: '2 hours ago',
      status: 'Completed',
    },
    {
      id: 'row-2',
      user: 'Mei Sato',
      task: 'Updated tax configuration',
      date: '5 hours ago',
      status: 'Review',
    },
    {
      id: 'row-3',
      user: 'Ken Ito',
      task: 'Queued invoice reminder batch',
      date: 'Yesterday',
      status: 'Pending',
    },
  ];

  const statusColors: Record<TableRow['status'], string> = {
    Completed: '#15803d',
    Pending: '#b45309',
    Review: '#1d4ed8',
  };

  const columns = [
    { header: 'User', accessor: userCell },
    { header: 'Action', accessor: actionCell },
    { header: 'Date', accessor: dateCell },
    { header: 'Status', accessor: statusCell },
  ] as const;
</script>

{#snippet userCell(row: TableRow)}
  <strong>{row.user}</strong>
{/snippet}

{#snippet actionCell(row: TableRow)}
  <span>{row.task}</span>
{/snippet}

{#snippet dateCell(row: TableRow)}
  <span style="color: #71717a;">{row.date}</span>
{/snippet}

{#snippet statusCell(row: TableRow)}
  <span
    style={`display: inline-flex; align-items: center; border-radius: 999px; padding: 4px 10px; background: color-mix(in srgb, ${statusColors[row.status]} 12%, white); color: ${statusColors[row.status]}; font-size: 0.75rem; font-weight: 700;`}
  >
    {row.status}
  </span>
{/snippet}

<div style="width: min(100%, 960px);">
  <Table columns={columns} data={rows} getRowKey={(row) => row.id} />
</div>
