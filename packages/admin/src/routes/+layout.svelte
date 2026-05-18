<script lang="ts">
	import type { Snippet } from 'svelte';

	import AdminShell from '$lib/components/layout/AdminShell.svelte';
	import '../app.css';

	interface LayoutOperator { id: string; email: string; role: string }
	const { children, data }: { children: Snippet; data: { operator: LayoutOperator | null; csrfToken: string; currentPath: string; navItems: { label: string; href: string; activePrefix: string }[] } } = $props();
</script>

<svelte:head>
	<title>Admin Console</title>
</svelte:head>


<div class="min-h-screen bg-slate-50 text-slate-950">
	{#if data.operator !== null}
		<AdminShell
			role={data.operator.role}
			currentPath={data.currentPath}
			navItems={data.navItems}
			operatorName={data.operator.email}
			csrfToken={data.csrfToken}
		>
			{@render children()}
		</AdminShell>
	{:else}
	{@render children()}
	{/if}
</div>
