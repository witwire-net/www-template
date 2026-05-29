<script lang="ts">
	import { getAdminAccountDetail } from '@www-template/admin-domain';
	import { Badge, Button, CardNS, ConfirmDialog, Input, Label, Separator } from '@www-template/ui/components';

	import { page } from '$app/state';

	import PasskeyList from '$lib/components/accounts/PasskeyList.svelte';
	import { createCurrentAdminI18n } from '$lib/i18n';

	interface AccountDetail {
		id: string;
		email: string;
		status: string;
		passkeyCount: number;
		createdAt: string;
	}

	interface PasskeyDetail {
		id: string;
		credentialHandle: string;
		createdAt: Date;
	}

	interface AccountDetailData { account: AccountDetail; passkeys: PasskeyDetail[]; csrfToken: string }

	const { data, form } = $props<{
		data?: Partial<AccountDetailData>;
		form?: { messageKey?: string };
	}>();
	let account = $state<AccountDetail | null>(null);
	let detailMessage = $state<string | null>(null);
	const pageData = $derived<AccountDetailData>({
		account: account ?? data?.account ?? {
			id: page.params.id ?? 'unknown',
			email: 'unknown@example.com',
			status: 'active',
			passkeyCount: 0,
			createdAt: new Date(0).toISOString(),
		},
		passkeys: data?.passkeys ?? [],
		csrfToken: data?.csrfToken ?? '',
	});
	const i18n = $derived(createCurrentAdminI18n());

	let suspendReason = $state('');
	let restoreReason = $state('');
	let lifecycleMessage = $state<string | null>(null);
	const passkeys = $derived(pageData.passkeys.map((passkey: PasskeyDetail) => ({ id: passkey.id, credential_handle: passkey.credentialHandle, created_at: i18n.formatDateTime(passkey.createdAt) })));
	const passkeyLabels = $derived({
		title: i18n.t('passkeyList.title'),
		emptyTitle: i18n.t('passkeyList.emptyTitle'),
		emptyDescription: i18n.t('passkeyList.emptyDescription'),
		badge: i18n.t('passkeyList.badge'),
		delete: i18n.t('passkeyList.delete'),
		add: i18n.t('passkeyList.add'),
	});

	$effect(() => {
		// detail route では server load を使わず、route param を domain function へ渡して Admin API detail を取得する。
		void loadAccountDetail(page.params.id ?? '');
	});

	async function loadAccountDetail(accountId: string): Promise<void> {
		// accountId が変わるたびに前回 error を消し、現在 route の結果だけを表示する。
		detailMessage = null;
		const result = await getAdminAccountDetail(accountId);
		if (!result.success) {
			detailMessage = accountErrorMessage(result.error);
			return;
		}

		// Admin API read model を detail 表示用 state に反映し、passkey 数も metadata として表示する。
		account = {
			id: result.data.id,
			email: result.data.email,
			status: result.data.status,
			passkeyCount: result.data.passkeyCount,
			createdAt: result.data.createdAt,
		};
	}

	function submitSuspend(): void {
		// 停止 mutation は後続の Admin domain/API layer で Authorization と CSRF を付与して実装する。
		lifecycleMessage = i18n.t('accountDetail.suspendDescription');
	}

	function submitRestore(): void {
		// 復旧 mutation も SvelteKit action ではなく Go Admin API 呼び出しへ移行する。
		lifecycleMessage = i18n.t('accountDetail.restoreDescription');
	}

	function accountErrorMessage(error: string): string {
		// domain error 分類だけを i18n 表示へ変換し、auth/session の詳細理由は隠す。
		if (error === 'unauthenticated') return i18n.t('accounts.errorUnauthenticated');
		if (error === 'forbidden') return i18n.t('accounts.errorForbidden');
		if (error === 'invalid-input') return i18n.t('accounts.errorInvalid');
		if (error === 'not-found') return i18n.t('accounts.errorNotFound');
		if (error === 'unavailable') return i18n.t('accounts.errorUnavailable');
		return i18n.t('accounts.errorUnknown');
	}

	function accountStatusLabel(statusValue: string): string {
		// lifecycle status は backend enum を保持しつつ、表示は i18n 辞書を通す。
		if (statusValue === 'active') return i18n.t('accounts.active');
		if (statusValue === 'suspended') return i18n.t('accounts.suspended');
		return statusValue;
	}
</script>

<svelte:head>
	<title>{pageData.account.email}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
		<div class="space-y-2">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('accountDetail.eyebrow')}</p>
			<h1 class="text-3xl font-bold tracking-tight">{pageData.account.email}</h1>
			<Badge variant={pageData.account.status === 'active' ? 'success' : 'danger'}>{accountStatusLabel(pageData.account.status)}</Badge>
		</div>
		<div class="flex gap-2">
			{#if pageData.account.status === 'active'}
				<ConfirmDialog title={i18n.t('accountDetail.suspendTitle')} description={i18n.t('accountDetail.suspendDescription')} confirmText={i18n.t('accountDetail.suspend')} confirmVariant="destructive" onConfirm={submitSuspend}>
					{#snippet trigger()}<Button variant="destructive">{i18n.t('accountDetail.suspend')}</Button>{/snippet}
				</ConfirmDialog>
			{:else}
				<ConfirmDialog title={i18n.t('accountDetail.restoreTitle')} description={i18n.t('accountDetail.restoreDescription')} confirmText={i18n.t('accountDetail.restore')} onConfirm={submitRestore}>
					{#snippet trigger()}<Button>{i18n.t('accountDetail.restore')}</Button>{/snippet}
				</ConfirmDialog>
			{/if}
		</div>
	</section>

	{#if detailMessage !== null}<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{detailMessage}</p>{/if}
	{#if form?.messageKey != null}<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{i18n.t(form.messageKey)}</p>{/if}
	{#if lifecycleMessage !== null}<p class="rounded-md border border-warning/20 bg-warning/10 p-3 text-sm text-warning">{lifecycleMessage}</p>{/if}

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{i18n.t('accountDetail.lifecycleTitle')}</CardNS.CardTitle>
			<CardNS.CardDescription>{i18n.t('accountDetail.lifecycleDescription')}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="grid gap-4 md:grid-cols-2">
			<div class="space-y-2">
				<Label for="suspend-reason">{i18n.t('accountDetail.suspendReason')}</Label>
				<Input id="suspend-reason" placeholder={i18n.t('accountDetail.suspendPlaceholder')} bind:value={suspendReason} />
			</div>
			<div class="space-y-2">
				<Label for="restore-reason">{i18n.t('accountDetail.restoreReason')}</Label>
				<Input id="restore-reason" bind:value={restoreReason} />
			</div>
		</CardNS.CardContent>
	</CardNS.Card>

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{i18n.t('accountDetail.metadataTitle')}</CardNS.CardTitle>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-3 text-sm text-foreground">
			<p>ID: {pageData.account.id}</p>
			<p>{i18n.t('passkeyList.title')}: {pageData.account.passkeyCount}</p>
			<Separator />
			<p>{i18n.t('accountDetail.created')} {i18n.formatDateTime(pageData.account.createdAt)}</p>
		</CardNS.CardContent>
	</CardNS.Card>

	<PasskeyList {passkeys} labels={passkeyLabels} />
</main>
