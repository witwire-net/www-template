<svelte:options runes={true} />

<script lang="ts">
  import Skeleton from '@ui/components/atoms/Skeleton/Skeleton.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import AppSidebar from '@ui/components/navigation/AppSidebar/AppSidebar.svelte';
  import AppHeader from '@ui/components/navigation/Header/AppHeader.svelte';
  import Table from '@ui/components/organisms/Table/Table.svelte';
  import DashboardLayout from '@ui/layouts/DashboardLayout/DashboardLayout.svelte';

  import styles from './Dashboard.module.scss';

  type ActivityRow = {
    id: number;
  };

  const sidebarLinks = [
    { label: 'Overview', href: '/dashboard/overview', active: true },
    { label: 'Analytics', href: '/dashboard/analytics' },
    { label: 'Customers', href: '/dashboard/customers' },
    { label: 'Settings', href: '/dashboard/settings' },
  ];

  const statItems = [1, 2, 3, 4] as const;
  const activityRows: readonly ActivityRow[] = [1, 2, 3, 4, 5].map((id) => ({ id }));
  const activityColumns = [
    { header: 'User', accessor: userCell },
    { header: 'Action', accessor: actionCell },
    { header: 'Date', accessor: dateCell },
    { header: 'Status', accessor: statusCell },
  ] as const;

  let loading = $state(true);
  let isSidebarOpen = $state(false);

  $effect(() => {
    const timer = window.setTimeout(() => {
      loading = false;
    }, 2000);

    return () => {
      window.clearTimeout(timer);
    };
  });
</script>

{#snippet userCell()}
  {#if loading}
    <Skeleton width={100} />
  {:else}
    John Doe
  {/if}
{/snippet}

{#snippet actionCell()}
  {#if loading}
    <Skeleton width={150} />
  {:else}
    Created new project
  {/if}
{/snippet}

{#snippet dateCell()}
  {#if loading}
    <Skeleton width={80} />
  {:else}
    2 hours ago
  {/if}
{/snippet}

{#snippet statusCell()}
  {#if loading}
    <Skeleton width={60} />
  {:else}
    <span class={styles.statusCompleted ?? ''}>Completed</span>
  {/if}
{/snippet}

{#snippet headerActions()}
  <div class={styles.headerActions ?? ''}>
    <span>Welcome, User</span>
    <div class={styles.userAvatar ?? ''}></div>
  </div>
{/snippet}

{#snippet sidebar()}
  <AppSidebar
    isOpen={isSidebarOpen}
    links={sidebarLinks}
    onClose={() => {
      isSidebarOpen = false;
    }}
  />
{/snippet}

{#snippet header()}
  <AppHeader
    logo={null}
    links={[]}
    actions={headerActions}
    onMenuClick={() => {
      isSidebarOpen = true;
    }}
  />
{/snippet}

<DashboardLayout {header} {sidebar}>
  <div class={styles.content ?? ''}>
    <h1 class={styles.pageTitle ?? ''}>Dashboard Overview</h1>

    <div class={styles.statsGrid ?? ''}>
      {#each statItems as item (item)}
        <Card className={styles.statCard ?? ''}>
          <CardBody>
            <div class={styles.statLabel ?? ''}>Total Revenue</div>
            {#if loading}
              <Skeleton variant="text" width="60%" height={36} />
            {:else}
              <div class={styles.statValue ?? ''}>$45,231</div>
            {/if}
          </CardBody>
        </Card>
      {/each}
    </div>

    <div class={styles.recentActivity ?? ''}>
      <div class={styles.tableScroll ?? ''}>
        <Table columns={activityColumns} data={activityRows} />
      </div>
    </div>
  </div>
</DashboardLayout>
