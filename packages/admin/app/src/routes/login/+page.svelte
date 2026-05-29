<script lang="ts">
	import { startAuthentication } from '@simplewebauthn/browser';
	import { finishAdminLogin, startAdminLogin } from '@www-template/admin-domain';
	import type { AdminLoginStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';

	import { createCurrentAdminI18n } from '$lib/i18n';

	let email = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const i18n = $derived(createCurrentAdminI18n());

	async function handlePasskeyLogin(): Promise<void> {
		// 連打による challenge の多重発行を避け、画面上も処理中であることを明示する。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// challenge 発行は Admin domain function に委譲し、app 層から API wrapper / generated SDK を直接呼ばない。
			const startPayload = await startAdminLogin(email);
			if (startPayload === null) throw new Error('login-start-failed');

			// WebAuthn ceremony はブラウザ API に限定し、秘密鍵 material を JavaScript へ取り出さない。
			const assertion = await startAuthentication({ optionsJSON: toAuthenticationOptions(startPayload.options) });

			// finish は Admin domain function に委譲し、accessToken / CSRF token だけを memory session に保持する。
			const session = await finishAdminLogin(startPayload.requestId, assertion);
			if (session === null) throw new Error('login-finish-failed');
			message = i18n.t('login.verified');
			void goto('/');
		} catch {
			// unknown email / inactive / invalid passkey を同じ文言にし、operator enumeration を防ぐ。
			message = i18n.t('login.error');
		} finally {
			// 処理終了時は必ず loading を解除し、再試行できる状態へ戻す。
			isSubmitting = false;
		}
	}

	function toAuthenticationOptions(options: AdminLoginStartResult['options']): Parameters<typeof startAuthentication>[0]['optionsJSON'] {
		// Admin API の descriptor type は OpenAPI 上 string なので、browser API 用に public-key literal へ絞る。
		return {
			challenge: options.challenge,
			rpId: options.rpId,
			timeout: options.timeout,
			allowCredentials: options.allowCredentials?.map((credential) => ({
				...credential,
				type: 'public-key' as const,
				transports: credential.transports as AuthenticatorTransport[] | undefined,
			})),
			userVerification: options.userVerification,
		};
	}
</script>

<svelte:head>
	<title>{i18n.t('login.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-8 lg:grid-cols-2">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('login.eyebrow')}</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">{i18n.t('login.heading')}</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">{i18n.t('login.description')}</p>
		</div>

		<CardNS.Card class="border-border bg-card text-card-foreground">
			<CardNS.CardHeader>
				<CardNS.CardTitle>{i18n.t('login.cardTitle')}</CardNS.CardTitle>
				<CardNS.CardDescription>{i18n.t('login.cardDescription')}</CardNS.CardDescription>
			</CardNS.CardHeader>
			<CardNS.CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="admin-login-email">{i18n.t('login.emailLabel')}</Label>
					<Input id="admin-login-email" type="email" autocomplete="email" bind:value={email} disabled={isSubmitting} placeholder="operator@example.com" />
				</div>
				{#if message !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={isSubmitting || email.trim() === ''} onclick={handlePasskeyLogin}>
					{#if isSubmitting}
						<Spinner aria-hidden="true" />
						{i18n.t('login.submitting')}
					{:else}
						{i18n.t('login.submit')}
					{/if}
				</Button>
				<Button class="w-full" variant="ghost" href="/operator-setup">{i18n.t('login.setupToken')}</Button>
			</CardNS.CardFooter>
		</CardNS.Card>
	</section>
</main>
