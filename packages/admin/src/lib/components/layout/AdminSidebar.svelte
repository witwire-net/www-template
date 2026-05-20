<script lang="ts">
	import { Sidebar, Badge } from '@www-template/ui/components';

	const {
		role = 'viewer',
		currentPath = '/',
		navItems = [],
		brandLabel,
		closeLabel,
	}: {
		role?: string;
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix?: string }[];
		brandLabel: string;
		closeLabel: string;
	} = $props();

	const visibleItems = $derived(navItems);

	function isActive(href: string): boolean {
		// `/` は全 route に前方一致してしまうため、dashboard だけ完全一致で判定する。
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}
</script>

<Sidebar.Sidebar ariaLabel={brandLabel} closeLabel={closeLabel}>
	<Sidebar.SidebarHeader>
		<Sidebar.SidebarMenu>
			<Sidebar.SidebarMenuItem>
				<Sidebar.SidebarMenuButton size="lg">
					{brandLabel}
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
