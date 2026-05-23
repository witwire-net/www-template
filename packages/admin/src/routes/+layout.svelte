<script lang="ts">
	import type { Snippet } from 'svelte';

	import AdminShell from '$lib/components/layout/AdminShell.svelte';
	import '../app.css';

	interface LayoutOperator { id: string; email: string; role: string; locale: 'ja' | 'en' }
	interface LayoutLabels { title: string; brand: string; admin: string; operatorFallback: string; logout: string; close: string }
	const { children, data }: { children: Snippet; data: { operator: LayoutOperator | null; csrfToken: string; currentPath: string; locale: 'ja' | 'en'; labels: LayoutLabels; navItems: { label: string; href: string; activePrefix: string }[] } } = $props();
</script>

<svelte:head>
	<title>{data.labels.title}</title>
</svelte:head>


<div class="min-h-screen bg-background text-foreground">
	{#if data.operator !== null}
		<AdminShell
			currentPath={data.currentPath}
			navItems={data.navItems}
			labels={data.labels}
			brandLabel={data.labels.brand}
			operatorName={data.operator.email}
			csrfToken={data.csrfToken}
		>
			{@render children()}
		</AdminShell>
	{:else}
	{@render children()}
	{/if}
</div>
