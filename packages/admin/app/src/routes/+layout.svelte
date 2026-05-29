<script lang="ts">
	import { verifyProtectedAdminRoute } from '@www-template/admin-domain';

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
	let operator = $state<LayoutOperator | null>(null);
	let routeState = $state<'public' | 'checking' | 'authenticated' | 'blocked'>('checking');
	const labels = $derived<LayoutLabels>({
		title: data?.labels?.title ?? i18n.t('layout.title'),
		brand: data?.labels?.brand ?? i18n.t('layout.brand'),
		admin: data?.labels?.admin ?? i18n.t('header.admin'),
		operatorFallback: data?.labels?.operatorFallback ?? i18n.t('header.operatorFallback'),
		logout: data?.labels?.logout ?? i18n.t('header.logout'),
		close: data?.labels?.close ?? i18n.t('shared.close'),
	});
	const isPublicRoute = $derived(currentPath === '/login' || currentPath === '/setup' || currentPath === '/operator-setup');
	const navItems = $derived(data?.navItems ?? [
		{ label: i18n.t('nav.dashboard'), href: '/', activePrefix: '/' },
		{ label: i18n.t('nav.accounts'), href: '/accounts', activePrefix: '/accounts' },
		{ label: i18n.t('nav.audit'), href: '/audit', activePrefix: '/audit' },
		{ label: i18n.t('nav.settings'), href: '/settings', activePrefix: '/settings' },
	]);

	$effect(() => {
		// public route は operator session を要求せず、login/setup 画面を即時表示する。
		if (isPublicRoute) {
			routeState = 'public';
			operator = null;
			return;
		}

		// protected route は表示前に Admin domain function で current operator を検証する。
		const verifiedPath = currentPath;
		void verifyCurrentOperator(verifiedPath);
	});

	async function verifyCurrentOperator(verifiedPath: string): Promise<void> {
		// route 遷移ごとに checking に戻し、古い operator で protected content が一瞬表示されることを防ぐ。
		routeState = 'checking';
		const result = await verifyProtectedAdminRoute();

		// 非同期検証中に route が変わった場合は、古い結果を現在画面へ反映しない。
		if (verifiedPath !== currentPath) return;

		if (result.status !== 'authenticated') {
			operator = null;
			routeState = 'blocked';
			void goto('/login');
			return;
		}

		// backend 検証済み operator だけを shell に渡し、UI role 表示を authorization の代替にしない。
		operator = {
			id: result.session.operator.operatorId,
			email: result.session.operator.email,
			role: result.session.operator.role,
			locale: i18n.locale,
		};
		routeState = 'authenticated';
	}
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
