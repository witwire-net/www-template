<script lang="ts">
	import { useAdminAccounts } from '@www-template/admin-domain';
	import type { AdminAccountDomainError } from '@www-template/admin-domain';
	import { Button, CardNS, EmptyState, Field, Input, Label, Select, Separator, Spinner } from '@www-template/ui/components';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	import AccountCreateForm from '$lib/components/accounts/AccountCreateForm.svelte';
	import AccountTable from '$lib/components/accounts/AccountTable.svelte';
	import { createCurrentAdminI18n } from '$lib/i18n';

	const i18n = $derived(createCurrentAdminI18n());

	const adminAccounts = useAdminAccounts({
		readSearchParams: () => page.url.searchParams,
		navigateTo: (url) => { void goto(url); },
	});
	const statusLabel = $derived(adminAccounts.data.state.status === '' ? i18n.t('accounts.allStatuses') : accountStatusLabel(adminAccounts.data.state.status));
	const tableAccounts = $derived(adminAccounts.data.state.accounts.map((account) => ({ id: account.id, email: account.email, status: account.status, status_label: accountStatusLabel(account.status), created_at: i18n.formatDateTime(account.createdAt) })));
	const totalPages = $derived(adminAccounts.data.state.nextCursor === null ? adminAccounts.data.state.currentPage : adminAccounts.data.state.currentPage + 1);
	const listMessage = $derived(adminAccounts.data.state.listError === null ? null : accountErrorMessage(adminAccounts.data.state.listError));
	const createMessage = $derived(adminAccounts.data.state.createError === null ? null : accountErrorMessage(adminAccounts.data.state.createError));
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
	const createLabels = $derived({
		title: i18n.t('accounts.createTitle'),
		description: i18n.t('accounts.createDescription'),
		email: i18n.t('accounts.email'),
		locale: i18n.t('accounts.createLocale'),
		localeJa: i18n.t('accounts.localeJa'),
		localeEn: i18n.t('accounts.localeEn'),
		submit: i18n.t('accounts.createSubmit'),
		submitting: i18n.t('accounts.createSubmitting'),
	});

	function applyFilters(): void {
		// 検索条件の URL 正規化と移動は domain action に委譲する。
		adminAccounts.actions.applyFilters();
	}

	function changePage(pageNumber: number): void {
		// cursor pagination の URL 更新は domain action に委譲する。
		adminAccounts.actions.changePage(pageNumber);
	}

	async function submitCreateAccount(): Promise<void> {
		// Account 作成 I/O と成功遷移は domain action に委譲する。
		await adminAccounts.actions.submitCreateAccount();
	}

	function openAccount(id: string): void {
		// 一覧行選択の遷移先構築は domain action に委譲する。
		adminAccounts.actions.openAccount(id);
	}

	function accountErrorMessage(error: AdminAccountDomainError): string {
		// domain error 分類だけを i18n 文言へ変換し、backend の内部 reason は表示しない。
		if (error === 'unauthenticated') return i18n.t('accounts.errorUnauthenticated');
		if (error === 'forbidden') return i18n.t('accounts.errorForbidden');
		if (error === 'invalid-input') return i18n.t('accounts.errorInvalid');
		if (error === 'duplicate-email') return i18n.t('accounts.errorDuplicate');
		if (error === 'not-found') return i18n.t('accounts.errorNotFound');
		if (error === 'unavailable') return i18n.t('accounts.errorUnavailable');
		return i18n.t('accounts.errorUnknown');
	}

	function accountStatusLabel(statusValue: string): string {
		// backend enum の表示名は辞書経由に寄せ、未知値だけ監査用に raw 値を残す。
		if (statusValue === 'active') return i18n.t('accounts.active');
		if (statusValue === 'suspended') return i18n.t('accounts.suspended');
		return statusValue;
	}
</script>

<svelte:head>
	<title>{i18n.t('accounts.title')}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">{i18n.t('accounts.title')}</h1>
		<p class="text-muted-foreground">{i18n.t('accounts.description')}</p>
	</section>

	<AccountCreateForm bind:email={adminAccounts.data.state.createEmail} bind:locale={adminAccounts.data.state.createLocale} message={createMessage} isSubmitting={adminAccounts.data.state.isCreating} labels={createLabels} onSubmit={() => { void submitCreateAccount(); }} />

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{i18n.t('accounts.filtersTitle')}</CardNS.CardTitle>
			<CardNS.CardDescription>{i18n.t('accounts.found', { total: adminAccounts.data.state.accounts.length })}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<Field.FieldGroup class="grid gap-4 md:grid-cols-3">
				<Field.Field>
					<Label for="account-query">{i18n.t('accounts.email')}</Label>
					<Input id="account-query" placeholder="customer@example.com" bind:value={adminAccounts.data.state.query} />
				</Field.Field>
				<Field.Field>
					<Label for="account-status">{i18n.t('accounts.status')}</Label>
					<Select.Select type="single" value={adminAccounts.data.state.status} onValueChange={(next: string) => { adminAccounts.data.state.status = next; }}>
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
			{#if adminAccounts.data.state.isLoading}
				<div class="flex items-center gap-2 text-sm text-muted-foreground"><Spinner aria-hidden="true" />{i18n.t('accounts.createSubmitting')}</div>
			{:else if listMessage !== null}
				<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{listMessage}</p>
			{:else if tableAccounts.length === 0}
				<EmptyState title={i18n.t('accounts.emptyTitle')} description={i18n.t('accounts.emptyDescription')} />
			{:else}
				<AccountTable accounts={tableAccounts} labels={tableLabels} page={adminAccounts.data.state.currentPage} {totalPages} onSelect={openAccount} onPageChange={changePage} />
			{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
