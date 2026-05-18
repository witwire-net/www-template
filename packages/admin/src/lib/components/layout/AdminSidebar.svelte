<script lang="ts">
	import { Sidebar, Badge } from '@www-template/ui/components';

	interface NavItem {
		label: string;
		href: string;
		roles: string[];
	}

	const defaultNavItems: NavItem[] = [
		{ label: 'Dashboard', href: '/', roles: ['admin', 'operator', 'viewer'] },
		{ label: 'Accounts', href: '/accounts', roles: ['admin', 'operator', 'viewer'] },
		{ label: 'Audit Log', href: '/audit', roles: ['admin', 'operator', 'viewer'] },
		{ label: 'Settings', href: '/settings', roles: ['admin'] },
	];

	const {
		role = 'viewer',
		currentPath = '/',
		navItems,
	}: {
		role?: string;
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix?: string }[];
	} = $props();

	const visibleItems = $derived(
		navItems ?? defaultNavItems.filter((item) => item.roles.includes(role))
	);

	function isActive(href: string): boolean {
		// `/` は全 route に前方一致してしまうため、dashboard だけ完全一致で判定する。
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}
</script>

<Sidebar.Sidebar>
	<Sidebar.SidebarHeader>
		<Sidebar.SidebarMenu>
			<Sidebar.SidebarMenuItem>
				<Sidebar.SidebarMenuButton size="lg">
					Admin Console
				</Sidebar.SidebarMenuButton>
			</Sidebar.SidebarMenuItem>
		</Sidebar.SidebarMenu>
	</Sidebar.SidebarHeader>
	<Sidebar.SidebarContent>
		<Sidebar.SidebarGroup>
			<Sidebar.SidebarGroupContent>
				<Sidebar.SidebarMenu>
					{#each visibleItems as item (item.href)}
						<Sidebar.SidebarMenuItem>
							<Sidebar.SidebarMenuLink
								isActive={isActive(item.href)}
								href={item.href}
							>
								{item.label}
							</Sidebar.SidebarMenuLink>
						</Sidebar.SidebarMenuItem>
					{/each}
				</Sidebar.SidebarMenu>
			</Sidebar.SidebarGroupContent>
		</Sidebar.SidebarGroup>
	</Sidebar.SidebarContent>
	<Sidebar.SidebarFooter>
		<Sidebar.SidebarMenu>
			<Sidebar.SidebarMenuItem>
				<Badge variant="outline">{role}</Badge>
			</Sidebar.SidebarMenuItem>
		</Sidebar.SidebarMenu>
	</Sidebar.SidebarFooter>
</Sidebar.Sidebar>
