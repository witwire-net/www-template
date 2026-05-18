<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';

	import { Button, Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, Input, Label, Spinner } from '@www-template/ui/components';

	interface RegistrationStartResponse {
		challengeId: string;
		options: Parameters<typeof startRegistration>[0];
	}

	let setupToken = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);

	async function handleOperatorSetup(): Promise<void> {
		// one-time token の多重消費を避けるため、登録処理中は再送信を止める。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// token の妥当性検証と challenge 作成は既存 BFF に集中させる。
			const startResponse = await globalThis.fetch('/api/admin/auth/operator-setup/start', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ setupToken }),
			});
			if (!startResponse.ok) throw new Error('operator-setup-start-failed');
			const startPayload = (await startResponse.json()) as RegistrationStartResponse;

			// ブラウザの authenticator で新しい passkey を作成し、登録応答だけを送信する。
			const attestation = await startRegistration(startPayload.options);

			// finish route は token 消費と passkey 追加を同一 transaction で処理する。
			const finishResponse = await globalThis.fetch('/api/admin/auth/operator-setup/finish', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ challengeId: startPayload.challengeId, attestation }),
			});
			if (!finishResponse.ok) throw new Error('operator-setup-finish-failed');
			globalThis.location.assign('/');
		} catch {
			// token の存在や期限切れ理由を細かく出さず、攻撃者に状態差分を渡さない。
			message = 'セットアップを完了できませんでした。トークンの期限とパスキー登録操作を確認してください。';
		} finally {
			// 失敗後も安全に再試行できるよう loading を解除する。
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>Operator Setup</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-8 lg:grid-cols-[1fr_28rem]">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">One-time operator setup</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">招待された運用者を、安全に有効化。</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">管理者から受け取ったセットアップトークンを使い、この端末のパスキーを登録します。</p>
		</div>

		<Card class="border-border bg-card text-card-foreground">
			<CardHeader>
				<CardTitle>オペレーターセットアップ</CardTitle>
				<CardDescription>one-time token は登録完了時に消費されます。</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="operator-setup-token">セットアップトークン</Label>
					<Input id="operator-setup-token" type="password" autocomplete="one-time-code" bind:value={setupToken} disabled={isSubmitting} />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardContent>
			<CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={isSubmitting || setupToken.trim() === ''} onclick={handleOperatorSetup}>
					{#if isSubmitting}
						<Spinner />
						登録中…
					{:else}
						パスキーを登録して開始
					{/if}
				</Button>
				<Button class="w-full" variant="ghost" href="/login">ログインへ戻る</Button>
			</CardFooter>
		</Card>
	</section>
</main>
