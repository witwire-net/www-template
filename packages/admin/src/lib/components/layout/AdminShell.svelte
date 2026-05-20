<script lang="ts">
	import { Sidebar } from '@www-template/ui/components';

	import AdminHeader from './AdminHeader.svelte';
	import AdminSidebar from './AdminSidebar.svelte';

	import type { Snippet } from 'svelte';

	const {
		children,
		role = 'viewer',
		currentPath = '/',
		navItems = [],
		labels,
		brandLabel,
		operatorName = '',
		csrfToken = '',
	}: {
		children: Snippet;
		role?: string;
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix: string }[];
		labels: { admin: string; operatorFallback: string; logout: string; close: string };
		brandLabel: string;
		operatorName?: string;
		csrfToken?: string;
	} = $props();
</script>

	<Sidebar.SidebarProvider>
		<AdminSidebar {role} {currentPath} {navItems} {brandLabel} closeLabel={labels.close} />
		<Sidebar.SidebarInset>
			<AdminHeader {operatorName} {csrfToken} adminLabel={labels.admin} operatorFallback={labels.operatorFallback} logoutLabel={labels.logout} />
			{@render children()}
		</Sidebar.SidebarInset>
</Sidebar.SidebarProvider>
