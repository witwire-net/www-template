<script lang="ts">
	import { Badge, Button, CardNS, ConfirmDialog, Input, Label, Separator } from '@www-template/ui/components';

	import PasskeyList from '$lib/components/accounts/PasskeyList.svelte';
	import { createAdminI18n } from '$lib/i18n';

	interface AccountDetail {
		id: string;
		email: string;
		status: string;
		statusReason: string | null;
		statusUpdatedAt: Date | null;
		sessionRevokedAfter: Date | null;
		createdAt: Date;
	}

	interface PasskeyDetail {
		id: string;
		credentialHandle: string;
		createdAt: Date;
	}

	const { data, form } = $props<{
		data: { locale: 'ja' | 'en'; account: AccountDetail; passkeys: PasskeyDetail[]; csrfToken: string };
		form?: { messageKey?: string };
	}>();
	const i18n = $derived(createAdminI18n(data.locale));

	let suspendForm = $state<HTMLFormElement | null>(null);
	let restoreForm = $state<HTMLFormElement | null>(null);
	let suspendReason = $state('');
	let restoreReason = $state('restored by operator');
	const passkeys = $derived(data.passkeys.map((passkey: PasskeyDetail) => ({ id: passkey.id, credential_handle: passkey.credentialHandle, created_at: new Date(passkey.createdAt).toISOString() })));
	const passkeyLabels = $derived({
		title: i18n.t('passkeyList.title'),
		emptyTitle: i18n.t('passkeyList.emptyTitle'),
		emptyDescription: i18n.t('passkeyList.emptyDescription'),
		badge: i18n.t('passkeyList.badge'),
		delete: i18n.t('passkeyList.delete'),
		add: i18n.t('passkeyList.add'),
	});

	function submitSuspend(): void {
		// ConfirmDialog の承認後にのみ SvelteKit form action を実行する。
		suspendForm?.requestSubmit();
	}

	function submitRestore(): void {
		// 復旧操作も確認を通して誤操作を防ぐ。
		restoreForm?.requestSubmit();
	}
</script>

<svelte:head>
	<title>{data.account.email}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
		<div class="space-y-2">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('accountDetail.eyebrow')}</p>
			<h1 class="text-3xl font-bold tracking-tight">{data.account.email}</h1>
			<Badge variant={data.account.status === 'active' ? 'success' : 'danger'}>{data.account.status}</Badge>
		</div>
		<div class="flex gap-2">
			{#if data.account.status === 'active'}
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

	{#if form?.messageKey != null}<p class="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{i18n.t(form.messageKey)}</p>{/if}

	<form bind:this={suspendForm} method="POST" action="?/suspend" class="hidden">
		<Input type="hidden" name="_csrf" value={data.csrfToken} />
		<Input name="reason" bind:value={suspendReason} />
	</form>
	<form bind:this={restoreForm} method="POST" action="?/restore" class="hidden">
		<Input type="hidden" name="_csrf" value={data.csrfToken} />
		<Input name="reason" bind:value={restoreReason} />
	</form>

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
		<CardNS.CardContent class="space-y-3 text-sm text-slate-700">
			<p>ID: {data.account.id}</p>
			<p>{i18n.t('accountDetail.statusReason')} {data.account.statusReason ?? '-'}</p>
			<p>{i18n.t('accountDetail.statusUpdated')} {data.account.statusUpdatedAt === null ? '-' : new Date(data.account.statusUpdatedAt).toISOString()}</p>
			<p>{i18n.t('accountDetail.sessionRevokedAfter')} {data.account.sessionRevokedAfter === null ? '-' : new Date(data.account.sessionRevokedAfter).toISOString()}</p>
			<Separator />
			<p>{i18n.t('accountDetail.created')} {new Date(data.account.createdAt).toISOString()}</p>
		</CardNS.CardContent>
	</CardNS.Card>

	<PasskeyList {passkeys} labels={passkeyLabels} />
</main>
