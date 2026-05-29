<script lang="ts">
	import { Breadcrumb } from '@www-template/ui/components';

	const {
		adminLabel,
	}: {
		adminLabel: string;
	} = $props();
</script>


	<header class="flex min-h-14 items-center rounded-md border border-border bg-surface px-6 py-3">
	<Breadcrumb.Breadcrumb aria-label={adminLabel}>
		<Breadcrumb.BreadcrumbList>
			<Breadcrumb.BreadcrumbItem>
				<Breadcrumb.BreadcrumbLink href="/">{adminLabel}</Breadcrumb.BreadcrumbLink>
			</Breadcrumb.BreadcrumbItem>
		</Breadcrumb.BreadcrumbList>
	</Breadcrumb.Breadcrumb>
</header>
