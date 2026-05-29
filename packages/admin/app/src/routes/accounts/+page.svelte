<script lang="ts">
	import { createCustomerAccount, searchAdminAccounts } from '@www-template/admin-domain';
	import { Button, CardNS, EmptyState, Field, Input, Label, Select, Separator, Spinner } from '@www-template/ui/components';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	import AccountCreateForm from '$lib/components/accounts/AccountCreateForm.svelte';
	import AccountTable from '$lib/components/accounts/AccountTable.svelte';
	import { createCurrentAdminI18n } from '$lib/i18n';

	const i18n = $derived(createCurrentAdminI18n());

	let query = $state('');
	let status = $state('');
	let cursor = $state<string | null>(null);
	let nextCursor = $state<string | null>(null);
	let currentPage = $state(1);
	let accounts = $state<{ id: string; email: string; status: string; createdAt: string; passkeyCount: number }[]>([]);
	let isLoading = $state(false);
	let isCreating = $state(false);
	let listMessage = $state<string | null>(null);
	let createMessage = $state<string | null>(null);
	let createEmail = $state('');
	let createLocale = $state<'ja' | 'en'>('ja');

	$effect(() => {
		// URL search を唯一の検索状態 source とし、server load/action なしでも再読み込み時に条件を復元する。
		void loadAccountsFromUrl(page.url.searchParams);
	});
	const statusLabel = $derived(status === '' ? i18n.t('accounts.allStatuses') : accountStatusLabel(status));
	const tableAccounts = $derived(accounts.map((account) => ({ id: account.id, email: account.email, status: account.status, status_label: accountStatusLabel(account.status), created_at: i18n.formatDateTime(account.createdAt) })));
	const totalPages = $derived(nextCursor === null ? currentPage : currentPage + 1);
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

	async function loadAccountsFromUrl(params: URLSearchParams): Promise<void> {
		// SvelteKit server load を使わず、domain function 経由で Admin API の account list を取得する。
		query = params.get('query') ?? '';
		status = params.get('status') ?? '';
		cursor = params.get('cursor');
		currentPage = Number(params.get('page') ?? '1');
		isLoading = true;
		listMessage = null;

		try {
			// backend contract は email/cursor/limit を受けるため、status filter は backend 対応まで UI 側表示条件として残す。
			const result = await searchAdminAccounts({ email: query, cursor: cursor ?? undefined, limit: 20 });
			if (!result.success) {
				accounts = [];
				listMessage = accountErrorMessage(result.error);
				return;
			}

			// status filter が URL にある場合だけ表示を絞り、Account lifecycle の source of truth は backend response に保つ。
			accounts = status === '' ? result.data.accounts : result.data.accounts.filter((account) => account.status === status);
			nextCursor = result.data.nextCursor;
		} finally {
			// 成功・失敗に関わらず loading を解除し、再検索できる状態へ戻す。
			isLoading = false;
		}
	}

	function buildUrl(pageNumber: number, nextPageCursor: string | null = null): string {
		// 画面状態を URL に正規化し、再読み込みや共有でも同じ検索条件を復元できるようにする。
		const params: string[] = [];
		if (query !== '') params.push(`query=${encodeURIComponent(query)}`);
		if (status !== '') params.push(`status=${encodeURIComponent(status)}`);
		if (nextPageCursor !== null) params.push(`cursor=${encodeURIComponent(nextPageCursor)}`);
		params.push(`page=${encodeURIComponent(String(pageNumber))}`);
		return `/accounts?${params.join('&')}`;
	}

	function applyFilters(): void {
		void goto(buildUrl(1));
	}

	function changePage(pageNumber: number): void {
		// cursor pagination なので、次ページは backend が返した opaque cursor がある場合だけ進める。
		if (pageNumber > currentPage && nextCursor !== null) {
			void goto(buildUrl(pageNumber, nextCursor));
			return;
		}

		// 前ページは cursor history を保持しないため、先頭ページへ戻して過去 cursor の誤用を避ける。
		void goto(buildUrl(1));
	}

	async function submitCreateAccount(): Promise<void> {
		// Account 作成は page から domain function へ委譲し、app 層から Admin API wrapper を直接 import しない。
		if (isCreating) return;
		isCreating = true;
		createMessage = null;

		try {
			const result = await createCustomerAccount({ email: createEmail, locale: createLocale });
			if (!result.success) {
				createMessage = accountErrorMessage(result.error);
				return;
			}

			// 作成成功後は作成済み account の詳細へ遷移し、duplicate retry 時以外は入力を残さない。
			createEmail = '';
			createMessage = i18n.t('accounts.createdSuccess');
			void goto(`/accounts/${result.data.id}`);
		} finally {
			// backend validation / network failure のどちらでも form を再操作可能にする。
			isCreating = false;
		}
	}

	function openAccount(id: string): void {
		void goto(`/accounts/${id}`);
	}

	function accountErrorMessage(error: string): string {
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

	<AccountCreateForm bind:email={createEmail} bind:locale={createLocale} message={createMessage} isSubmitting={isCreating} labels={createLabels} onSubmit={() => { void submitCreateAccount(); }} />

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{i18n.t('accounts.filtersTitle')}</CardNS.CardTitle>
			<CardNS.CardDescription>{i18n.t('accounts.found', { total: accounts.length })}</CardNS.CardDescription>
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
			{#if isLoading}
				<div class="flex items-center gap-2 text-sm text-muted-foreground"><Spinner aria-hidden="true" />{i18n.t('accounts.createSubmitting')}</div>
			{:else if listMessage !== null}
				<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{listMessage}</p>
			{:else if tableAccounts.length === 0}
				<EmptyState title={i18n.t('accounts.emptyTitle')} description={i18n.t('accounts.emptyDescription')} />
			{:else}
				<AccountTable accounts={tableAccounts} labels={tableLabels} page={currentPage} {totalPages} onSelect={openAccount} onPageChange={changePage} />
			{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
