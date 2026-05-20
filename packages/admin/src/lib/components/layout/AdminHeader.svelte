<script lang="ts">
	import { Avatar, Breadcrumb, Button, DropdownMenu, Input } from '@www-template/ui/components';

	const {
		operatorName = '',
		csrfToken = '',
		adminLabel,
		operatorFallback,
		logoutLabel,
	}: {
		operatorName?: string;
		csrfToken?: string;
		adminLabel: string;
		operatorFallback: string;
		logoutLabel: string;
	} = $props();
</script>


	<header class="flex min-h-16 items-center justify-between border-b bg-white px-6 shadow-sm">
	<Breadcrumb.Breadcrumb aria-label={adminLabel}>
		<Breadcrumb.BreadcrumbList>
			<Breadcrumb.BreadcrumbItem>
				<Breadcrumb.BreadcrumbLink href="/">{adminLabel}</Breadcrumb.BreadcrumbLink>
			</Breadcrumb.BreadcrumbItem>
		</Breadcrumb.BreadcrumbList>
	</Breadcrumb.Breadcrumb>
	<DropdownMenu.DropdownMenu>
		<DropdownMenu.DropdownMenuTriggerButton variant="ghost" class="gap-2">
			<Avatar.Avatar class="h-6 w-6">
				<Avatar.AvatarFallback>{(operatorName !== '' ? operatorName : 'O')[0]}</Avatar.AvatarFallback>
			</Avatar.Avatar>
			{operatorName !== '' ? operatorName : operatorFallback}
		</DropdownMenu.DropdownMenuTriggerButton>
		<DropdownMenu.DropdownMenuContent align="end">
			<DropdownMenu.DropdownMenuLabel>{operatorName}</DropdownMenu.DropdownMenuLabel>
			<DropdownMenu.DropdownMenuSeparator />
			<form method="POST" action="/api/admin/auth/logout">
				<Input type="hidden" name="_csrf" value={csrfToken} />
				<Button type="submit" variant="ghost" class="w-full justify-start">{logoutLabel}</Button>
			</form>
		</DropdownMenu.DropdownMenuContent>
	</DropdownMenu.DropdownMenu>
</header>
