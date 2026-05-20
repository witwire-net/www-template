<script lang="ts">
	import { startAuthentication } from '@simplewebauthn/browser';

	import { Button, Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, Input, Label, Spinner } from '@www-template/ui/components';

	interface LoginLabels {
		title: string;
		eyebrow: string;
		heading: string;
		description: string;
		cardTitle: string;
		cardDescription: string;
		emailLabel: string;
		error: string;
		submitting: string;
		submit: string;
		setupToken: string;
	}

	interface LoginStartResponse {
		challengeId: string;
		options: Parameters<typeof startAuthentication>[0];
	}

	let email = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const { data } = $props<{ data: { labels: LoginLabels } }>();

	async function handlePasskeyLogin(): Promise<void> {
		// 連打による challenge の多重発行を避け、画面上も処理中であることを明示する。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// 既存の Admin BFF start route を使い、email の有無で応答 shape が変わらない flow を維持する。
			const startResponse = await globalThis.fetch('/api/admin/auth/passkey/start', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email }),
			});
			if (!startResponse.ok) throw new Error('login-start-failed');
			const startPayload = (await startResponse.json()) as LoginStartResponse;

			// WebAuthn ceremony はブラウザ API に限定し、秘密鍵 material を JavaScript へ取り出さない。
			const assertion = await startAuthentication(startPayload.options);

			// finish route が session cookie を発行するため、成功後は root へ遷移する。
			const finishResponse = await globalThis.fetch('/api/admin/auth/passkey/finish', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ challengeId: startPayload.challengeId, assertion }),
			});
			if (!finishResponse.ok) throw new Error('login-finish-failed');
			globalThis.location.assign('/');
		} catch {
			// unknown email / inactive / invalid passkey を同じ文言にし、operator enumeration を防ぐ。
			message = data.labels.error;
		} finally {
			// 処理終了時は必ず loading を解除し、再試行できる状態へ戻す。
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>{data.labels.title}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-8 lg:grid-cols-[1fr_28rem]">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{data.labels.eyebrow}</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">{data.labels.heading}</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">{data.labels.description}</p>
		</div>

		<Card class="border-border bg-card text-card-foreground">
			<CardHeader>
				<CardTitle>{data.labels.cardTitle}</CardTitle>
				<CardDescription>{data.labels.cardDescription}</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="admin-login-email">{data.labels.emailLabel}</Label>
					<Input id="admin-login-email" type="email" autocomplete="email" bind:value={email} disabled={isSubmitting} placeholder="operator@example.com" />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardContent>
			<CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={isSubmitting || email.trim() === ''} onclick={handlePasskeyLogin}>
					{#if isSubmitting}
						<Spinner aria-hidden="true" />
						{data.labels.submitting}
					{:else}
						{data.labels.submit}
					{/if}
				</Button>
				<Button class="w-full" variant="ghost" href="/operator-setup">{data.labels.setupToken}</Button>
			</CardFooter>
		</Card>
	</section>
</main>
