<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import AppSidebar from '@ui/components/navigation/AppSidebar/AppSidebar.svelte';
  import Header from '@ui/components/navigation/Header/Header.svelte';
  import Table from '@ui/components/organisms/Table/Table.svelte';

  import DashboardLayout from './DashboardLayout.svelte';

  type ActivityRow = {
    action: string;
    date: string;
    id: string;
    status: 'Completed' | 'In review';
    user: string;
  };

  let { rich = false }: { rich?: boolean } = $props();

  const dashboardLinks = [
    { label: 'Overview', href: '/dashboard/overview', active: true },
    { label: 'Analytics', href: '/dashboard/analytics' },
    { label: 'Customers', href: '/dashboard/customers' },
    { label: 'Settings', href: '/dashboard/settings' },
  ];

  const activityRows: readonly ActivityRow[] = [
    {
      id: 'activity-1',
      user: 'John Doe',
      action: 'Created new project',
      date: '2 hours ago',
      status: 'Completed',
    },
    {
      id: 'activity-2',
      user: 'Mei Sato',
      action: 'Updated billing profile',
      date: '5 hours ago',
      status: 'In review',
    },
    {
      id: 'activity-3',
      user: 'Aki Tanaka',
      action: 'Shared retention dashboard',
      date: 'Yesterday',
      status: 'Completed',
    },
  ];

  const statCards = [
    { label: 'Total Revenue', value: '$45,231' },
    { label: 'New Customers', value: '1,283' },
    { label: 'Active Trials', value: '246' },
    { label: 'NPS Score', value: '64' },
  ] as const;

  const statusColorMap: Record<ActivityRow['status'], string> = {
    Completed: '#15803d',
    'In review': '#b45309',
  };

  const tableColumns = [
    { header: 'User', accessor: userCell },
    { header: 'Action', accessor: actionCell },
    { header: 'Date', accessor: dateCell },
    { header: 'Status', accessor: statusCell },
  ] as const;

  let isSidebarOpen = $state(false);
  let lastAction = $state('None');
</script>

{#snippet userCell(row: ActivityRow)}
  <strong>{row.user}</strong>
{/snippet}

{#snippet actionCell(row: ActivityRow)}
  <span>{row.action}</span>
{/snippet}

{#snippet dateCell(row: ActivityRow)}
  <span style="color: var(--color-text-muted);">{row.date}</span>
{/snippet}

{#snippet statusCell(row: ActivityRow)}
  <span
    style={`display: inline-flex; align-items: center; border-radius: 999px; padding: 4px 10px; background: color-mix(in srgb, ${statusColorMap[row.status]} 12%, white); color: ${statusColorMap[row.status]}; font-size: 0.75rem; font-weight: 700;`}
  >
    {row.status}
  </span>
{/snippet}

{#snippet headerActions()}
  <div style="display: flex; align-items: center; gap: 16px;">
    <Button
      variant="ghost"
      size="sm"
      onclick={() => {
        lastAction = 'Notifications opened';
      }}
    >
      Notifications
    </Button>
    <div
      style="width: 32px; height: 32px; border-radius: 999px; background: var(--color-primary);"
    ></div>
  </div>
{/snippet}

{#snippet sidebarLogo()}
  <div
    style="display: flex; align-items: center; gap: 8px; color: var(--color-primary); padding: 0 0.5rem;"
  >
    <div style="width: 24px; height: 24px; background: currentColor; border-radius: 6px;"></div>
    <span style="font-size: 1.25rem; font-weight: 900;">www-template UI</span>
  </div>
{/snippet}

{#snippet sidebarFooter()}
  <div style="display: flex; flex-direction: column; gap: 0.5rem; padding: 0 0.5rem;">
    <div
      style="padding: 0.75rem; background: var(--color-surface-hover); border-radius: var(--radius-md); display: flex; align-items: center; gap: 8px;"
    >
      <div style="width: 32px; height: 32px; border-radius: 999px; background: var(--color-border);"></div>
      <div style="overflow: hidden;">
        <div style="font-size: 0.875rem; font-weight: 700;">John Doe</div>
        <div style="font-size: 0.75rem; color: var(--color-text-muted);">Admin</div>
      </div>
    </div>
    <Button
      variant="outline"
      size="sm"
      style="width: 100%;"
      onclick={() => {
        lastAction = 'Logged out';
      }}
    >
      Logout
    </Button>
  </div>
{/snippet}

<DashboardLayout>
  {#snippet header()}
    <Header
      variant="app"
      actions={rich ? headerActions : undefined}
      links={[]}
      onMenuClick={() => {
        isSidebarOpen = true;
      }}
    />
  {/snippet}

  {#snippet sidebar()}
    <AppSidebar
      footer={rich ? sidebarFooter : '© 2024 www-template UI'}
      isOpen={isSidebarOpen}
      items={dashboardLinks}
      header={rich ? sidebarLogo : 'www-template UI'}
      onClose={() => (isSidebarOpen = false)}
    />
  {/snippet}

  <div style="padding: 2rem;">
    {#if rich}
      <div style="padding: 0 0 1rem; color: var(--color-text-muted);">Last action: {lastAction}</div>
    {/if}

    <h1 style="margin-bottom: 2rem; font-size: 1.5rem; font-weight: 700;">Dashboard Overview</h1>

    <div
      style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1rem; margin-bottom: 2rem;"
    >
      {#each statCards as stat (stat.label)}
        <Card>
          <CardBody>
            <div style="margin-bottom: 0.5rem; font-size: 0.875rem; color: var(--color-text-muted);">{stat.label}</div>
            <div style="font-size: 1.5rem; font-weight: 700;">{stat.value}</div>
          </CardBody>
        </Card>
      {/each}
    </div>

    <div
      style="background: white; border: 1px solid var(--color-border-subtle); border-radius: 1rem; padding: 1.5rem;"
    >
      <h2 style="margin-bottom: 1rem; font-size: 1.125rem; font-weight: 700;">Recent Activity</h2>
      <Table columns={tableColumns} data={activityRows} getRowKey={(row) => row.id} />
    </div>
  </div>
</DashboardLayout>
