<script lang="ts">
	import { Button, CardNS, EmptyState, Field, Input, Label, Select, Separator } from '@www-template/ui/components';

	import { goto } from '$app/navigation';

	import AccountTable from '$lib/components/accounts/AccountTable.svelte';

	interface AccountRow {
		id: string;
		email: string;
		status: string;
		createdAt: Date;
	}

	interface PageShape {
		accounts: AccountRow[];
		page: number;
		totalPages: number;
		total: number;
		filters: { query: string; status: string };
	}

	const { data } = $props<{ data: PageShape }>();

	const filters = $derived(data.filters);
	let query = $state('');
	let status = $state('');

	$effect(() => {
		query = filters.query;
		status = filters.status;
	});
	const statusLabel = $derived(status === '' ? 'All statuses' : status);
	const tableAccounts = $derived(data.accounts.map((account: AccountRow) => ({ ...account, created_at: new Date(account.createdAt).toISOString() })));

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

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">Accounts</h1>
		<p class="text-slate-600">顧客アカウントを検索し、停止状態と passkey 数を確認します。</p>
	</section>

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>Search filters</CardNS.CardTitle>
			<CardNS.CardDescription>{data.total} accounts found</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<Field.FieldGroup class="grid gap-4 md:grid-cols-3">
				<Field.Field>
					<Label for="account-query">Email</Label>
					<Input id="account-query" placeholder="customer@example.com" bind:value={query} />
				</Field.Field>
				<Field.Field>
					<Label for="account-status">Status</Label>
					<Select.Select type="single" value={status} onValueChange={(next: string) => { status = next; }}>
						<Select.SelectTrigger id="account-status"><Select.SelectValue>{statusLabel}</Select.SelectValue></Select.SelectTrigger>
						<Select.SelectContent>
							<Select.SelectItem value="">All statuses</Select.SelectItem>
							<Select.SelectItem value="active">active</Select.SelectItem>
							<Select.SelectItem value="suspended">suspended</Select.SelectItem>
						</Select.SelectContent>
					</Select.Select>
				</Field.Field>
				<Field.Field class="justify-end">
					<Button onclick={applyFilters}>Apply filters</Button>
				</Field.Field>
			</Field.FieldGroup>
			<Separator />
			{#if tableAccounts.length === 0}
				<EmptyState title="No accounts found" description="検索条件に一致するアカウントはありません。" />
			{:else}
				<AccountTable accounts={tableAccounts} page={data.page} totalPages={data.totalPages} onSelect={openAccount} onPageChange={(page: number) => { void goto(buildUrl(page)); }} />
			{/if}
		</CardNS.CardContent>
	</CardNS.Card>
</main>
