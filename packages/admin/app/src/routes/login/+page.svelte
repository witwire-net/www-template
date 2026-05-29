<script lang="ts">
	import { startAuthentication } from '@simplewebauthn/browser';
	import { useAdminLogin } from '@www-template/admin-domain';
	import type { AdminLoginStartResult } from '@www-template/admin-domain';

	import { Button, CardNS, Input, Label, Spinner } from '@www-template/ui/components';
	import { goto } from '$app/navigation';

	import { createCurrentAdminI18n } from '$lib/i18n';

	const login = useAdminLogin();
	const i18n = $derived(createCurrentAdminI18n());

	async function handlePasskeyLogin(): Promise<void> {
		// WebAuthn browser API と navigation だけを app 層 callback として渡し、認証 I/O は domain action に委譲する。
		await login.actions.submit(
			(options) => startAuthentication({ optionsJSON: toAuthenticationOptions(options) }),
			() => { void goto('/'); }
		);
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
					<Input id="admin-login-email" type="email" autocomplete="email" bind:value={login.data.state.email} disabled={login.data.state.isSubmitting} placeholder="operator@example.com" />
				</div>
				{#if login.data.state.messageKey !== null}
					<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{i18n.t(login.data.state.messageKey)}</p>
				{/if}
			</CardNS.CardContent>
			<CardNS.CardFooter class="flex flex-col gap-3">
				<Button class="w-full" size="lg" disabled={login.data.state.isSubmitting || login.data.state.email.trim() === ''} onclick={handlePasskeyLogin}>
					{#if login.data.state.isSubmitting}
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
