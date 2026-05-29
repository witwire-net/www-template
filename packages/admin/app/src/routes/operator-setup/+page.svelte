<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { finishOperatorSetup, startOperatorSetup } from '@www-template/admin-domain';
	import type { AdminOperatorSetupStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	import { createCurrentAdminI18n } from '$lib/i18n';

	let setupToken = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const i18n = $derived(createCurrentAdminI18n());
	let consumedTokenFromUrl = $state(false);

	$effect.pre(() => {
		// 配送メールの `/operator-setup?token=...` から one-time token を一度だけ取り込み、手入力と同じ form state に移す。
		const tokenFromUrl = page.url.searchParams.get('token')?.trim() ?? '';
		if (consumedTokenFromUrl || tokenFromUrl === '') return;
		setupToken = tokenFromUrl;
		consumedTokenFromUrl = true;

		// token 平文を browser history / address bar に残さないため、入力へ取り込んだ直後に query を削除する。
		const sanitizedUrl = new URL(page.url);
		sanitizedUrl.searchParams.delete('token');
		globalThis.history.replaceState(globalThis.history.state, '', `${sanitizedUrl.pathname}${sanitizedUrl.search}${sanitizedUrl.hash}`);
	});

	async function handleOperatorSetup(): Promise<void> {
		// one-time token の多重消費を避けるため、登録処理中は再送信を止める。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// token の妥当性検証と challenge 作成は Admin domain function 経由で Go Admin API へ委譲する。
			const startPayload = await startOperatorSetup(setupToken);
			if (startPayload === null) throw new Error('operator-setup-start-failed');

			// ブラウザの authenticator で新しい passkey を作成し、登録応答だけを送信する。
			const attestation = await startRegistration({ optionsJSON: toRegistrationOptions(startPayload.options) });

			// finish route は token 消費と passkey 追加を backend transaction へ委譲し、session state だけを memory に保持する。
			const session = await finishOperatorSetup(setupToken, startPayload.requestId, attestation);
			if (session === null) throw new Error('operator-setup-finish-failed');
			void goto('/');
		} catch {
			// token の存在や期限切れ理由を細かく出さず、攻撃者に状態差分を渡さない。
			message = i18n.t('operatorSetup.error');
		} finally {
			// 失敗後も安全に再試行できるよう loading を解除する。
			isSubmitting = false;
		}
	}

	function toRegistrationOptions(options: AdminOperatorSetupStartResult['options']): Parameters<typeof startRegistration>[0]['optionsJSON'] {
		// Admin API の rpId/rpName read model を SimpleWebAuthn browser が要求する rp object へ変換する。
		return {
			challenge: options.challenge,
			rp: { id: options.rpId, name: options.rpName },
			user: options.user,
			pubKeyCredParams: options.pubKeyCredParams.map((parameter) => ({
				...parameter,
				type: 'public-key' as const,
			})),
			timeout: options.timeout,
			excludeCredentials: options.excludeCredentials?.map((credential) => ({
				...credential,
				type: 'public-key' as const,
				transports: credential.transports as AuthenticatorTransport[] | undefined,
			})),
			authenticatorSelection: {
				residentKey: options.residentKey,
				requireResidentKey: options.requireResidentKey,
				userVerification: options.userVerification,
			},
			attestation: options.attestation as AttestationConveyancePreference | undefined,
		};
	}
</script>

<svelte:head>
	<title>{i18n.t('operatorSetup.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-8 lg:grid-cols-2">
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
