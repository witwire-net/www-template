<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { useAdminInitialSetup } from '@www-template/admin-domain';
	import type { AdminInitialSetupStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';

	import { createCurrentAdminI18n } from '$lib/i18n';

	type SetupAvailability = 'available' | 'operator-exists' | 'bootstrap-disabled';
	type InitialSetupStartedResult = Extract<AdminInitialSetupStartResult, { status: 'started' }>;

	const { data } = $props<{
		data?: { setupAvailability?: SetupAvailability };
	}>();
	const i18n = $derived(createCurrentAdminI18n());
	const initialSetup = useAdminInitialSetup({ readInitialAvailability: () => data?.setupAvailability });
	const showSetupForm = $derived(initialSetup.data.state.setupAvailability === 'available');
	const unavailableMessage = $derived(
		initialSetup.data.state.setupAvailability === 'operator-exists'
			? i18n.t('setup.operatorExists')
			: i18n.t('setup.unavailable')
	);

	async function handleInitialSetup(): Promise<void> {
		// WebAuthn 登録と navigation だけを app 層 callback として渡し、初回 setup I/O は domain action に委譲する。
		await initialSetup.actions.submit(
			(options) => startRegistration({ optionsJSON: toRegistrationOptions(options) }),
			() => { void goto('/'); }
		);
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
						<Input id="setup-email" type="email" autocomplete="email" bind:value={initialSetup.data.state.email} disabled={initialSetup.data.state.isSubmitting} placeholder="admin@example.com" />
					</div>
					<div class="space-y-2">
						<Label for="setup-display-name">{i18n.t('setup.displayName')}</Label>
						<Input id="setup-display-name" autocomplete="name" bind:value={initialSetup.data.state.displayName} disabled={initialSetup.data.state.isSubmitting} />
					</div>
					<div class="space-y-2">
						<Label for="setup-secret">{i18n.t('setup.secret')}</Label>
						<Input id="setup-secret" type="password" autocomplete="one-time-code" bind:value={initialSetup.data.state.bootstrapSecret} disabled={initialSetup.data.state.isSubmitting} />
					</div>
					{#if initialSetup.data.state.messageKey !== null}
						<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{i18n.t(initialSetup.data.state.messageKey)}</p>
					{/if}
				</CardNS.CardContent>
				<CardNS.CardFooter>
					<Button class="w-full" size="lg" disabled={initialSetup.data.state.isSubmitting || initialSetup.data.state.email.trim() === '' || initialSetup.data.state.displayName.trim() === '' || initialSetup.data.state.bootstrapSecret.trim() === ''} onclick={handleInitialSetup}>
						{#if initialSetup.data.state.isSubmitting}
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
