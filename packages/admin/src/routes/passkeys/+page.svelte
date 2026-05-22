<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';

	import { Button, CardNS, EmptyState, Spinner, ConfirmDialog } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	interface PasskeyItem {
		id: string;
		createdAt: string;
		backupEligible: boolean;
		backupState: boolean;
		transports: unknown;
	}

	interface RegistrationStartResponse {
		challengeId: string;
		options: Parameters<typeof startRegistration>[0]['optionsJSON'];
	}

	interface PasskeyListResponse {
		passkeys: PasskeyItem[];
	}

	const { data }: { data: { locale: 'ja' | 'en'; operator: { email: string }; passkeys: PasskeyItem[]; csrfToken: string } } = $props();
	const i18n = $derived(createAdminI18n(data.locale));

	let passkeys = $state(initialPasskeys());
	let isAdding = $state(false);
	let deletingId = $state<string | null>(null);
	let pendingDeleteId = $state<string | null>(null);
	let message = $state<string | null>(null);
	let deleteDialogOpen = $state(false);

	const csrfToken: string | undefined = $derived(data.csrfToken !== '' ? data.csrfToken : undefined);


	function initialPasskeys(): PasskeyItem[] {
		// server load 由来の初期一覧を、後続の追加・削除で置き換え可能な local state に移す。
		return data.passkeys;
	}

	function formatDate(iso: string): string {
		// DB の ISO 文字列を operator locale に合わせた日時へ整形する。
		return new Date(iso).toLocaleString(i18n.locale === 'en' ? 'en-US' : 'ja-JP', { dateStyle: 'medium', timeStyle: 'short' });
	}

	async function refreshPasskeys(): Promise<void> {
		// 追加・削除後は BFF から最新 metadata を取得し、画面と server load cache を揃える。
		const response = await globalThis.fetch('/api/admin/auth/passkeys');
		if (!response.ok) throw new Error('passkey-refresh-failed');
		const payload = (await response.json()) as PasskeyListResponse;
		passkeys = payload.passkeys;
	}

	async function handleAddPasskey(): Promise<void> {
		// 追加処理中は重複 challenge を作らないようボタン操作を止める。
		if (isAdding) return;
		isAdding = true;
		message = null;

		try {
			// 認証済み BFF は session-bound CSRF header を要求するため、cookie 値を同一 origin のみへ送る。
			const startResponse = await globalThis.fetch('/api/admin/auth/passkeys/start', {
				method: 'POST',
				headers: csrfToken !== undefined ? { 'x-csrf-token': csrfToken } : {},
			});
			if (!startResponse.ok) throw new Error('passkey-add-start-failed');
			const startPayload = (await startResponse.json()) as RegistrationStartResponse;

			// authenticator で新しい passkey を作成し、attestation だけを finish route に渡す。
			const attestation = await startRegistration({ optionsJSON: startPayload.options });

			// 追加完了時は BFF が credential を検証し、公開 metadata のみを返す。
			const finishResponse = await globalThis.fetch('/api/admin/auth/passkeys/finish', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', ...(csrfToken !== undefined ? { 'x-csrf-token': csrfToken } : {}) },
				body: JSON.stringify({ challengeId: startPayload.challengeId, attestation }),
			});
			if (!finishResponse.ok) throw new Error('passkey-add-finish-failed');
			await refreshPasskeys();
			message = i18n.t('passkeys.addSuccess');
		} catch {
			// WebAuthn cancel / CSRF / server error の詳細を分けず、安全な再試行メッセージにする。
			message = i18n.t('passkeys.addError');
		} finally {
			// 成功・失敗にかかわらず追加操作を再度行えるようにする。
			isAdding = false;
		}
	}

	function requestDelete(passkeyId: string): void {
		// 破壊的操作は即時実行せず、ConfirmDialog の確認を必ず挟む。
		pendingDeleteId = passkeyId;
		deleteDialogOpen = true;
	}

	async function confirmDelete(): Promise<void> {
		// dialog 確認時点の対象 ID を固定し、途中で state が変わっても別 passkey を削除しない。
		const targetId = pendingDeleteId;
		if (targetId === null) return;
		deletingId = targetId;
		message = null;

		try {
			// DELETE は本人所有 passkey だけを BFF が削除し、最後の passkey は server 側で拒否する。
			const response = await globalThis.fetch(`/api/admin/auth/passkeys/${encodeURIComponent(targetId)}`, {
				method: 'DELETE',
				headers: csrfToken !== undefined ? { 'x-csrf-token': csrfToken } : {},
			});
			if (!response.ok) throw new Error('passkey-delete-failed');
			await refreshPasskeys();
			message = i18n.t('passkeys.deleteSuccess');
		} catch {
			// 削除失敗時は対象有無や最後の passkey かどうかを必要以上に露出しない。
			message = i18n.t('passkeys.deleteError');
		} finally {
			// dialog の対象と loading をクリアし、次の操作に備える。
			pendingDeleteId = null;
			deletingId = null;
		}
	}
</script>

<svelte:head>
	<title>{i18n.t('passkeys.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-10 text-foreground">
	<section class="mx-auto max-w-5xl space-y-8">
		<div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
			<div class="space-y-2">
				<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('passkeys.eyebrow')}</p>
				<h1 class="text-4xl font-black tracking-tight">{i18n.t('passkeys.heading')}</h1>
				<p class="text-sm text-muted-foreground">{i18n.t('passkeys.description', { email: data.operator.email })}</p>
			</div>
				<Button disabled={isAdding} onclick={handleAddPasskey}>
					{#if isAdding}
						<Spinner aria-hidden="true" />
						{i18n.t('passkeys.adding')}
					{:else}
					{i18n.t('passkeys.add')}
				{/if}
			</Button>
		</div>

		{#if message !== null}
			<p class="rounded-2xl border border-border bg-card px-4 py-3 text-sm text-card-foreground" role="status">{message}</p>
		{/if}

		<CardNS.Card>
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('passkeys.registeredTitle')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('passkeys.registeredDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent>
				{#if passkeys.length === 0}
					<EmptyState title={i18n.t('passkeys.emptyTitle')} description={i18n.t('passkeys.emptyDescription')} />
				{:else}
					<div class="space-y-3">
						{#each passkeys as passkey, index (passkey.id)}
							<div class="flex flex-col gap-4 rounded-3xl border border-border bg-card p-4 md:flex-row md:items-center md:justify-between">
								<div class="space-y-1">
									<p class="text-sm font-bold">{i18n.t('passkeys.passkey')} {index + 1}</p>
									<p class="text-xs text-muted-foreground">{i18n.t('passkeys.credentialId')} {passkey.id}</p>
									<p class="text-sm text-muted-foreground">{i18n.t('passkeys.createdAt')} {formatDate(passkey.createdAt)}</p>
									<p class="text-xs text-muted-foreground">{i18n.t('passkeys.backup')} {passkey.backupEligible ? (passkey.backupState ? i18n.t('passkeys.backupSynced') : i18n.t('passkeys.backupEligible')) : i18n.t('passkeys.backupDeviceBound')}</p>
								</div>
							<Button variant="destructive" disabled={passkeys.length <= 1 || deletingId === passkey.id} onclick={() => { requestDelete(passkey.id); }}>
								{#if deletingId === passkey.id}
									<Spinner aria-hidden="true" />
								{i18n.t('passkeys.deleting')}
								{:else}
									{i18n.t('passkeys.delete')}
									{/if}
								</Button>
							</div>
						{/each}
					</div>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter>
				<Button variant="outline" href="/">{i18n.t('passkeys.back')}</Button>
			</CardNS.CardFooter>
		</CardNS.Card>
	</section>
</main>

<ConfirmDialog
	bind:open={deleteDialogOpen}
	title={i18n.t('passkeys.deleteTitle')}
	description={i18n.t('passkeys.deleteDescription')}
	confirmText={i18n.t('passkeys.deleteConfirm')}
	cancelText={i18n.t('passkeys.cancel')}
	confirmVariant="destructive"
	onConfirm={confirmDelete}
/>
