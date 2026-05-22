<script lang="ts">
	import { Button, CardNS, EmptyState, Field, Input, Label, Select, Separator } from '@www-template/ui/components';

	import { goto } from '$app/navigation';

	import AccountTable from '$lib/components/accounts/AccountTable.svelte';
	import { createAdminI18n } from '$lib/i18n';

	interface AccountRow {
		id: string;
		email: string;
		status: string;
		createdAt: Date;
	}

	interface PageShape {
		locale: 'ja' | 'en';
		accounts: AccountRow[];
		page: number;
		totalPages: number;
		total: number;
		filters: { query: string; status: string };
	}

	const { data } = $props<{ data: PageShape }>();
	const i18n = $derived(createAdminI18n(data.locale));

	const filters = $derived(data.filters);
	let query = $state('');
	let status = $state('');

	$effect(() => {
		query = filters.query;
		status = filters.status;
	});
	const statusLabel = $derived(status === '' ? i18n.t('accounts.allStatuses') : status);
	const tableAccounts = $derived(data.accounts.map((account: AccountRow) => ({ ...account, created_at: new Date(account.createdAt).toISOString() })));
	const tableLabels = $derived({
		caption: i18n.t('accounts.tableCaption'),
		email: i18n.t('accounts.email'),
		status: i18n.t('accounts.status'),
		created: i18n.t('accounts.created'),
		actions: i18n.t('accounts.actions'),
		view: i18n.t('accounts.view'),
		pagination: i18n.t('shared.pagination'),
		previousPage: i18n.t('shared.previousPage'),
		nextPage: i18n.t('shared.nextPage'),
	});

	function buildUrl(page: number): string {
		// 画面状態を URL に正規化し、再読み込みや共有でも同じ検索条件を復元できるようにする。
		const params: string[] = [];
		if (query !== '') params.push(`query=${encodeURIComponent(query)}`);
		if (status !== '') params.push(`status=${encodeURIComponent(status)}`);
		params.push(`page=${encodeURIComponent(String(page))}`);
		return `/accounts?${params.join('&')}`;
	}

	function applyFilters(): void {
		void goto(buildUrl(1));
	}

	function openAccount(id: string): void {
		void goto(`/accounts/${id}`);
	}
</script>

<svelte:head>
	<title>{i18n.t('accounts.title')} - Admin Console</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">{i18n.t('accounts.title')}</h1>
		<p class="text-slate-600">{i18n.t('accounts.description')}</p>
	</section>

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{i18n.t('accounts.filtersTitle')}</CardNS.CardTitle>
			<CardNS.CardDescription>{i18n.t('accounts.found', { total: data.total })}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<Field.FieldGroup class="grid gap-4 md:grid-cols-3">
				<Field.Field>
					<Label for="account-query">{i18n.t('accounts.email')}</Label>
					<Input id="account-query" placeholder="customer@example.com" bind:value={query} />
				</Field.Field>
				<Field.Field>
					<Label for="account-status">{i18n.t('accounts.status')}</Label>
					<Select.Select type="single" value={status} onValueChange={(next: string) => { status = next; }}>
						<Select.SelectTrigger id="account-status"><Select.SelectValue>{statusLabel}</Select.SelectValue></Select.SelectTrigger>
						<Select.SelectContent>
							<Select.SelectItem value="">{i18n.t('accounts.allStatuses')}</Select.SelectItem>
							<Select.SelectItem value="active">{i18n.t('accounts.active')}</Select.SelectItem>
							<Select.SelectItem value="suspended">{i18n.t('accounts.suspended')}</Select.SelectItem>
						</Select.SelectContent>
					</Select.Select>
				</Field.Field>
				<Field.Field class="justify-end">
					<Button onclick={applyFilters}>{i18n.t('accounts.applyFilters')}</Button>
				</Field.Field>
			</Field.FieldGroup>
			<Separator />
			{#if tableAccounts.length === 0}
				<EmptyState title={i18n.t('accounts.emptyTitle')} description={i18n.t('accounts.emptyDescription')} />
			{:else}
				<AccountTable accounts={tableAccounts} labels={tableLabels} page={data.page} totalPages={data.totalPages} onSelect={openAccount} onPageChange={(page: number) => { void goto(buildUrl(page)); }} />
			{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
