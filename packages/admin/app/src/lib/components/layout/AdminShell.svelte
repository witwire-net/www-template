<script lang="ts">
	import { Sidebar } from '@www-template/ui/components';

	import AdminHeader from './AdminHeader.svelte';
	import AdminSidebar from './AdminSidebar.svelte';

	import type { Snippet } from 'svelte';

	const {
		children,
		currentPath = '/',
		navItems = [],
		labels,
		brandLabel,
		operatorName = '',
	}: {
		children: Snippet;
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix: string }[];
		labels: { admin: string; operatorFallback: string; logout: string; close: string };
		brandLabel: string;
		operatorName?: string;
	} = $props();
</script>

	<Sidebar.SidebarProvider>
		<AdminSidebar {currentPath} {navItems} {brandLabel} closeLabel={labels.close} {operatorName} operatorFallback={labels.operatorFallback} logoutLabel={labels.logout} />
		<Sidebar.SidebarInset class="gap-4 p-3">
			<AdminHeader adminLabel={labels.admin} />
			{@render children()}
		</Sidebar.SidebarInset>
</Sidebar.SidebarProvider>
