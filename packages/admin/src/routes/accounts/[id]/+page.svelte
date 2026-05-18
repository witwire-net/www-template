<script lang="ts">
	import { Badge, Button, CardNS, ConfirmDialog, Input, Label, Separator } from '@www-template/ui/components';

	import PasskeyList from '$lib/components/accounts/PasskeyList.svelte';

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
		data: { account: AccountDetail; passkeys: PasskeyDetail[]; csrfToken: string };
		form?: { message?: string };
	}>();

	let suspendForm = $state<HTMLFormElement | null>(null);
	let restoreForm = $state<HTMLFormElement | null>(null);
	let suspendReason = $state('');
	let restoreReason = $state('Restored after operator review');
	const passkeys = $derived(data.passkeys.map((passkey: PasskeyDetail) => ({ id: passkey.id, credential_handle: passkey.credentialHandle, created_at: new Date(passkey.createdAt).toISOString() })));

	function submitSuspend(): void {
		// ConfirmDialog の承認後にのみ SvelteKit form action を実行する。
		suspendForm?.requestSubmit();
	}

	function submitRestore(): void {
		// 復旧操作も確認を通して誤操作を防ぐ。
		restoreForm?.requestSubmit();
	}
</script>

<main class="space-y-6 p-8">
	<section class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
		<div class="space-y-2">
			<p class="text-sm font-semibold uppercase tracking-wide text-slate-500">Account detail</p>
			<h1 class="text-3xl font-bold tracking-tight">{data.account.email}</h1>
			<Badge variant={data.account.status === 'active' ? 'success' : 'danger'}>{data.account.status}</Badge>
		</div>
		<div class="flex gap-2">
			{#if data.account.status === 'active'}
				<ConfirmDialog title="Suspend account" description="この顧客アカウントを停止します。" confirmText="Suspend" confirmVariant="destructive" onConfirm={submitSuspend}>
					{#snippet trigger()}<Button variant="destructive">Suspend</Button>{/snippet}
				</ConfirmDialog>
			{:else}
				<ConfirmDialog title="Restore account" description="この顧客アカウントを復旧します。" confirmText="Restore" onConfirm={submitRestore}>
					{#snippet trigger()}<Button>Restore</Button>{/snippet}
				</ConfirmDialog>
			{/if}
		</div>
	</section>

	{#if form?.message != null}<p class="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{form.message}</p>{/if}

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
			<CardNS.CardTitle>Lifecycle controls</CardNS.CardTitle>
			<CardNS.CardDescription>停止・復旧の理由は監査ログへ保存されます。</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="grid gap-4 md:grid-cols-2">
			<div class="space-y-2">
				<Label for="suspend-reason">Suspend reason</Label>
				<Input id="suspend-reason" placeholder="Abuse investigation" bind:value={suspendReason} />
			</div>
			<div class="space-y-2">
				<Label for="restore-reason">Restore reason</Label>
				<Input id="restore-reason" bind:value={restoreReason} />
			</div>
		</CardNS.CardContent>
	</CardNS.Card>

	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>Account metadata</CardNS.CardTitle>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-3 text-sm text-slate-700">
			<p>ID: {data.account.id}</p>
			<p>Status reason: {data.account.statusReason ?? '-'}</p>
			<p>Status updated: {data.account.statusUpdatedAt === null ? '-' : new Date(data.account.statusUpdatedAt).toISOString()}</p>
			<p>Session revoked after: {data.account.sessionRevokedAfter === null ? '-' : new Date(data.account.sessionRevokedAfter).toISOString()}</p>
			<Separator />
			<p>Created: {new Date(data.account.createdAt).toISOString()}</p>
		</CardNS.CardContent>
	</CardNS.Card>

	<PasskeyList {passkeys} />
</main>
