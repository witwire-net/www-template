<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	interface RegistrationStartResponse {
		challengeId: string;
		options: Parameters<typeof startRegistration>[0]['optionsJSON'];
	}

	let setupToken = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const { data } = $props<{ data: { locale: 'ja' | 'en' } }>();
	const i18n = $derived(createAdminI18n(data.locale));

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
			const attestation = await startRegistration({ optionsJSON: startPayload.options });

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
			message = i18n.t('operatorSetup.error');
		} finally {
			// 失敗後も安全に再試行できるよう loading を解除する。
			isSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>{i18n.t('operatorSetup.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-8 lg:grid-cols-[1fr_28rem]">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('operatorSetup.eyebrow')}</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">{i18n.t('operatorSetup.heading')}</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">{i18n.t('operatorSetup.description')}</p>
		</div>

		<CardNS.Card class="border-border bg-card text-card-foreground">
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('operatorSetup.cardTitle')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('operatorSetup.cardDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="operator-setup-token">{i18n.t('operatorSetup.token')}</Label>
					<Input id="operator-setup-token" type="password" autocomplete="one-time-code" bind:value={setupToken} disabled={isSubmitting} />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={isSubmitting || setupToken.trim() === ''} onclick={handleOperatorSetup}>
						{#if isSubmitting}
							<Spinner aria-hidden="true" />
							{i18n.t('operatorSetup.submitting')}
						{:else}
						{i18n.t('operatorSetup.submit')}
					{/if}
				</Button>
				<Button class="w-full" variant="ghost" href="/login">{i18n.t('operatorSetup.backToLogin')}</Button>
			</CardNS.CardFooter>
		</CardNS.Card>
	</section>
</main>
