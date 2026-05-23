<script lang="ts">
	import { Avatar, Button, DropdownMenu, Input, Sidebar } from '@www-template/ui/components';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import UsersIcon from '@lucide/svelte/icons/users';
	import ClipboardListIcon from '@lucide/svelte/icons/clipboard-list';
	import SettingsIcon from '@lucide/svelte/icons/settings';

	const {
		currentPath = '/',
		navItems = [],
		brandLabel,
		closeLabel,
		operatorName = '',
		operatorFallback,
		logoutLabel,
		csrfToken = '',
	}: {
		currentPath?: string;
		navItems?: { label: string; href: string; activePrefix?: string }[];
		brandLabel: string;
		closeLabel: string;
		operatorName?: string;
		operatorFallback: string;
		logoutLabel: string;
		csrfToken?: string;
	} = $props();

	const visibleItems = $derived(navItems);

	function isActive(href: string): boolean {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}

	const iconMap: Record<string, typeof LayoutDashboardIcon> = {
		'/': LayoutDashboardIcon,
		'/accounts': UsersIcon,
		'/audit': ClipboardListIcon,
		'/settings': SettingsIcon,
	};

	const operatorDisplayName = $derived(operatorName !== '' ? operatorName : operatorFallback);
</script>

<Sidebar.Sidebar variant="floating" ariaLabel={brandLabel} closeLabel={closeLabel}>
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
								{@const Icon = iconMap[item.href] ?? LayoutDashboardIcon}
								<Icon />
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
				<DropdownMenu.DropdownMenu>
					<DropdownMenu.DropdownMenuTriggerButton variant="ghost" class="h-auto w-full justify-start gap-2 rounded-sm px-3 py-2 text-left">
						<Avatar.Avatar class="h-6 w-6">
							<Avatar.AvatarFallback>{operatorDisplayName[0]}</Avatar.AvatarFallback>
						</Avatar.Avatar>
						<span class="truncate">{operatorDisplayName}</span>
					</DropdownMenu.DropdownMenuTriggerButton>
					<DropdownMenu.DropdownMenuContent align="start" class="w-56">
						<DropdownMenu.DropdownMenuLabel>{operatorDisplayName}</DropdownMenu.DropdownMenuLabel>
						<DropdownMenu.DropdownMenuSeparator />
						<form method="POST" action="/api/admin/auth/logout">
							<Input type="hidden" name="_csrf" value={csrfToken} />
							<Button type="submit" variant="ghost" class="h-auto w-full justify-start rounded-sm px-3 py-2 text-sm">{logoutLabel}</Button>
						</form>
					</DropdownMenu.DropdownMenuContent>
				</DropdownMenu.DropdownMenu>
			</Sidebar.SidebarMenuItem>
		</Sidebar.SidebarMenu>
	</Sidebar.SidebarFooter>
</Sidebar.Sidebar>
