<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { finishInitialAdminSetup, startInitialAdminSetup } from '@www-template/admin-domain';
	import type { AdminInitialSetupStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';

	import { createCurrentAdminI18n } from '$lib/i18n';

	type SetupAvailability = 'available' | 'operator-exists' | 'bootstrap-disabled';
	type InitialSetupStartedResult = Extract<AdminInitialSetupStartResult, { status: 'started' }>;

	let email = $state('');
	let displayName = $state('');
	let bootstrapSecret = $state('');
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);
	const { data } = $props<{
		data?: { setupAvailability?: SetupAvailability };
	}>();
	const i18n = $derived(createCurrentAdminI18n());
	let setupAvailability = $state<SetupAvailability>('available');
	const showSetupForm = $derived(setupAvailability === 'available');
	const unavailableMessage = $derived(
		setupAvailability === 'operator-exists'
			? i18n.t('setup.operatorExists')
			: i18n.t('setup.unavailable')
	);

	$effect.pre(() => {
		// test fixture や将来の runtime state が availability を渡す場合だけ、form 表示可否へ反映する。
		if (data?.setupAvailability !== undefined) setupAvailability = data.setupAvailability;
	});

	async function handleInitialSetup(): Promise<void> {
		// 初回管理者作成は二重送信を防ぎ、transaction 側の競合検知に過度に頼らない。
		if (isSubmitting) return;
		isSubmitting = true;
		message = null;

		try {
			// bootstrap secret 検証と challenge 発行は Admin domain 経由で Go Admin API へ委譲する。
			const startPayload = await startInitialAdminSetup({ email, displayName, bootstrapSecret });
			if (startPayload.status !== 'started') {
				// operator 既存や bootstrap gate 無効を form 非表示 state へ反映し、secret 入力欄を残さない。
				if (startPayload.status === 'operator-exists') setupAvailability = 'operator-exists';
				if (startPayload.status === 'bootstrap-disabled') setupAvailability = 'bootstrap-disabled';
				throw new Error('initial-setup-start-failed');
			}

			// browser WebAuthn API で最初の admin passkey を作成し、秘密鍵 material を JS へ露出しない。
			const attestation = await startRegistration({
				optionsJSON: toRegistrationOptions(startPayload.options),
			});

			// finish は Admin backend に operator/passkey/session 作成を委譲し、accessToken だけを memory state に残す。
			const session = await finishInitialAdminSetup(
				{ email, displayName, bootstrapSecret },
				startPayload.requestId,
				attestation
			);
			if (session === null) throw new Error('initial-setup-finish-failed');
			void goto('/');
		} catch {
			// bootstrap secret や operator 件数の詳細を出さず、初回 setup の状態推測を防ぐ。
			message = i18n.t('setup.error');
		} finally {
			// 成功・失敗にかかわらず loading を戻し、画面操作を復帰させる。
			isSubmitting = false;
		}
	}

	function toRegistrationOptions(
		options: InitialSetupStartedResult['options']
	): Parameters<typeof startRegistration>[0]['optionsJSON'] {
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
	<title>{i18n.t('setup.title')}</title>
</svelte:head>

<main class="min-h-screen bg-background px-6 py-12 text-foreground">
	<section class="mx-auto grid min-h-screen max-w-5xl items-center gap-10 lg:grid-cols-2">
		<div class="space-y-6">
			<p class="text-sm font-semibold uppercase tracking-widest text-muted-foreground">{i18n.t('setup.eyebrow')}</p>
			<h1 class="max-w-2xl text-4xl font-black tracking-tight text-foreground md:text-6xl">{i18n.t('setup.heading')}</h1>
			<p class="max-w-xl text-base leading-7 text-muted-foreground">{i18n.t('setup.description')}</p>
		</div>

		{#if showSetupForm}
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
						<Input id="setup-display-name" autocomplete="name" bind:value={displayName} disabled={isSubmitting} />
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
		{:else}
			<CardNS.Card class="border-border bg-card text-card-foreground">
				<CardNS.CardHeader>
					<CardNS.CardTitle>{i18n.t('setup.unavailableTitle')}</CardNS.CardTitle>
					<CardNS.CardDescription>{unavailableMessage}</CardNS.CardDescription>
				</CardNS.CardHeader>
				<CardNS.CardFooter>
					<Button class="w-full" variant="ghost" href="/login">{i18n.t('operatorSetup.backToLogin')}</Button>
				</CardNS.CardFooter>
			</CardNS.Card>
		{/if}
	</section>
</main>
