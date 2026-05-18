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
		operatorName = '',
		csrfToken = '',
	}: {
		children: Snippet;
		role?: string;
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix: string }[];
		operatorName?: string;
		csrfToken?: string;
	} = $props();
</script>

<Sidebar.SidebarProvider>
	<AdminSidebar {role} {currentPath} {navItems} />
	<Sidebar.SidebarInset>
		<AdminHeader {operatorName} {csrfToken} />
		{@render children()}
	</Sidebar.SidebarInset>
</Sidebar.SidebarProvider>
