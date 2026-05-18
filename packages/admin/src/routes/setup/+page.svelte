<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';

	import { Button, Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, Input, Label, Spinner } from '@www-template/ui/components';

	interface RegistrationStartResponse {
		challengeId: string;
		options: Parameters<typeof startRegistration>[0];
	}

	let email = $state('');
	let displayName = $state('');
	let bootstrapSecret = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);

	async function handleInitialSetup(): Promise<void> {
		// 初回管理者作成は二重送信を防ぎ、transaction 側の競合検知に過度に頼らない。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// bootstrap secret は BFF へ直接渡し、画面 state 以外に保存しない。
			const startResponse = await globalThis.fetch('/api/admin/auth/setup/start', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, displayName, bootstrapSecret }),
			});
			if (!startResponse.ok) throw new Error('setup-start-failed');
			const startPayload = (await startResponse.json()) as RegistrationStartResponse;

			// Passkey credential はブラウザの WebAuthn ceremony で作成し、秘密鍵をサーバーへ送らない。
			const attestation = await startRegistration(startPayload.options);

			// finish route が初回 operator と passkey を同一 transaction で保存する。
			const finishResponse = await globalThis.fetch('/api/admin/auth/setup/finish', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ challengeId: startPayload.challengeId, attestation }),
			});
			if (!finishResponse.ok) throw new Error('setup-finish-failed');
			globalThis.location.assign('/');
		} catch {
			// bootstrap secret や WebAuthn 失敗の詳細を出さず、再試行可能な案内に留める。
			message = '初回セットアップを完了できませんでした。入力内容とパスキー登録操作を確認してください。';
		} finally {
			// 成功・失敗にかかわらず loading を戻し、画面操作を復帰させる。
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>Admin Bootstrap</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-6xl items-center gap-10 lg:grid-cols-[1.1fr_30rem]">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">First operator bootstrap</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">最初の管理者だけが、管理境界を作る。</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">まだ Admin operator が存在しない環境でのみ使用できます。bootstrap secret と最初のパスキーを登録してください。</p>
		</div>

		<Card class="border-border bg-card text-card-foreground">
			<CardHeader>
				<CardTitle>初回セットアップ</CardTitle>
				<CardDescription>最初の admin operator とログイン用パスキーを作成します。</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="setup-email">メールアドレス</Label>
					<Input id="setup-email" type="email" autocomplete="email" bind:value={email} disabled={isSubmitting} placeholder="admin@example.com" />
				</div>
				<div class="space-y-2">
					<Label for="setup-display-name">表示名</Label>
					<Input id="setup-display-name" autocomplete="name" bind:value={displayName} disabled={isSubmitting} placeholder="Admin Operator" />
				</div>
				<div class="space-y-2">
					<Label for="setup-secret">Bootstrap secret</Label>
					<Input id="setup-secret" type="password" autocomplete="one-time-code" bind:value={bootstrapSecret} disabled={isSubmitting} />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardContent>
			<CardFooter>
				<Button class="w-full" size="lg" disabled={isSubmitting || email.trim() === '' || displayName.trim() === '' || bootstrapSecret.trim() === ''} onclick={handleInitialSetup}>
					{#if isSubmitting}
						<Spinner />
						登録中…
					{:else}
						最初のパスキーを登録
					{/if}
				</Button>
			</CardFooter>
		</Card>
	</section>
</main>
