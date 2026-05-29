<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { useAdminOperatorSetup } from '@www-template/admin-domain';
	import type { AdminOperatorSetupStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	import { createCurrentAdminI18n } from '$lib/i18n';

	const i18n = $derived(createCurrentAdminI18n());
	const operatorSetup = useAdminOperatorSetup({
		readUrl: () => page.url,
		replaceUrl: (url) => {
			// token 平文を browser history / address bar に残さないため、domain が作った sanitized URL だけを反映する。
			globalThis.history.replaceState(globalThis.history.state, '', url);
		},
	});

	async function handleOperatorSetup(): Promise<void> {
		// WebAuthn 登録と navigation だけを app 層 callback として渡し、token I/O は domain action に委譲する。
		await operatorSetup.actions.submit(
			(options) => startRegistration({ optionsJSON: toRegistrationOptions(options) }),
			() => { void goto('/'); }
		);
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
					<Input id="operator-setup-token" type="password" autocomplete="one-time-code" bind:value={operatorSetup.data.state.setupToken} disabled={operatorSetup.data.state.isSubmitting} />
				</div>
				{#if operatorSetup.data.state.messageKey !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{i18n.t(operatorSetup.data.state.messageKey)}</p>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={operatorSetup.data.state.isSubmitting || operatorSetup.data.state.setupToken.trim() === ''} onclick={handleOperatorSetup}>
						{#if operatorSetup.data.state.isSubmitting}
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
