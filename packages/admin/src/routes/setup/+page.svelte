<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	interface RegistrationStartResponse {
		challengeId: string;
		options: Parameters<typeof startRegistration>[0]['optionsJSON'];
	}

	let email = $state('');
	let displayName = $state('');
	let bootstrapSecret = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const { data } = $props<{ data: { locale: 'ja' | 'en' } }>();
	const i18n = $derived(createAdminI18n(data.locale));

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
			const attestation = await startRegistration({ optionsJSON: startPayload.options });

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
			message = i18n.t('setup.error');
		} finally {
			// 成功・失敗にかかわらず loading を戻し、画面操作を復帰させる。
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>{i18n.t('setup.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-10 lg:grid-cols-[1.1fr_30rem]">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('setup.eyebrow')}</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">{i18n.t('setup.heading')}</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">{i18n.t('setup.description')}</p>
		</div>

		<CardNS.Card class="border-border bg-card text-card-foreground">
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('setup.cardTitle')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('setup.cardDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="setup-email">{i18n.t('setup.email')}</Label>
					<Input id="setup-email" type="email" autocomplete="email" bind:value={email} disabled={isSubmitting} placeholder="admin@example.com" />
				</div>
				<div class="space-y-2">
					<Label for="setup-display-name">{i18n.t('setup.displayName')}</Label>
					<Input id="setup-display-name" autocomplete="name" bind:value={displayName} disabled={isSubmitting} placeholder="Admin Operator" />
				</div>
				<div class="space-y-2">
					<Label for="setup-secret">{i18n.t('setup.secret')}</Label>
					<Input id="setup-secret" type="password" autocomplete="one-time-code" bind:value={bootstrapSecret} disabled={isSubmitting} />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter>
				<Button class="w-full" size="lg" disabled={isSubmitting || email.trim() === '' || displayName.trim() === '' || bootstrapSecret.trim() === ''} onclick={handleInitialSetup}>
						{#if isSubmitting}
							<Spinner aria-hidden="true" />
							{i18n.t('setup.submitting')}
						{:else}
						{i18n.t('setup.submit')}
					{/if}
				</Button>
			</CardNS.CardFooter>
		</CardNS.Card>
	</section>
</main>
