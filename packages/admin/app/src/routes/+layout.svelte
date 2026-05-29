<script lang="ts">
	import { useAdminSession } from '@www-template/admin-domain';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';

	import AdminShell from '$lib/components/layout/AdminShell.svelte';
	import { createCurrentAdminI18n } from '$lib/i18n';
	import '../app.css';

	interface LayoutOperator { id: string; email: string; role: string; locale: 'ja' | 'en' }
	interface LayoutLabels { title: string; brand: string; admin: string; operatorFallback: string; logout: string; close: string }
	interface LayoutData { operator: LayoutOperator | null; csrfToken: string; currentPath: string; labels: LayoutLabels; navItems: { label: string; href: string; activePrefix: string }[] }

	const { children, data }: { children: Snippet; data?: Partial<LayoutData> } = $props();
	const currentPath = $derived(data?.currentPath ?? page.url.pathname);
	const i18n = $derived(createCurrentAdminI18n());
	const session = useAdminSession({
		readPath: () => page.url.pathname,
		isPublicPath: (path) => path === '/login' || path === '/setup' || path === '/operator-setup',
		redirectToLogin: () => { void goto('/login'); },
	});
	const operator = $derived<LayoutOperator | null>(session.data.state.session === null ? null : {
		id: session.data.state.session.operator.operatorId,
		email: session.data.state.session.operator.email,
		role: session.data.state.session.operator.role,
		locale: i18n.locale,
	});
	const routeState = $derived(session.data.state.routeState);
	const labels = $derived<LayoutLabels>({
		title: data?.labels?.title ?? i18n.t('layout.title'),
		brand: data?.labels?.brand ?? i18n.t('layout.brand'),
		admin: data?.labels?.admin ?? i18n.t('header.admin'),
		operatorFallback: data?.labels?.operatorFallback ?? i18n.t('header.operatorFallback'),
		logout: data?.labels?.logout ?? i18n.t('header.logout'),
		close: data?.labels?.close ?? i18n.t('shared.close'),
	});
	const navItems = $derived(data?.navItems ?? [
		{ label: i18n.t('nav.dashboard'), href: '/', activePrefix: '/' },
		{ label: i18n.t('nav.accounts'), href: '/accounts', activePrefix: '/accounts' },
		{ label: i18n.t('nav.audit'), href: '/audit', activePrefix: '/audit' },
		{ label: i18n.t('nav.settings'), href: '/settings', activePrefix: '/settings' },
	]);

</script>

<svelte:head>
	<title>{labels.title}</title>
</svelte:head>


<div class="min-h-screen bg-background text-foreground">
	{#if routeState === 'checking'}
		<div class="flex min-h-screen items-center justify-center text-sm text-muted-foreground">{labels.operatorFallback}</div>
	{:else if operator !== null}
		<AdminShell
			{currentPath}
			{navItems}
			{labels}
			brandLabel={labels.brand}
			operatorName={operator.email}
		>
			{@render children()}
		</AdminShell>
	{:else if routeState === 'public'}
		{@render children()}
	{:else if routeState === 'blocked'}
		<div class="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
			<a class="underline underline-offset-4" href="/login">{labels.operatorFallback}</a>
		</div>
	{/if}
</div>
