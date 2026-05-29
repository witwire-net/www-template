<script lang="ts">
	import { useAdminSettings } from '@www-template/admin-domain';
	import { Button, CardNS, Label, Select, Separator } from '@www-template/ui/components';

	import { createCurrentAdminI18n, getCurrentAdminLocale, setCurrentAdminLocale } from '$lib/i18n';

	interface SettingsLabels {
		title: string;
		description: string;
		languageTitle: string;
		languageDescription: string;
		languageLabel: string;
		languageJapanese: string;
		languageEnglish: string;
		languageSubmit: string;
		languageSuccess: string;
		languageError: string;
		managementTitle: string;
		managementDescription: string;
		managementBody: string;
		managementButton: string;
	}

	const { data } = $props<{ data?: { canManageOperators?: boolean; operatorCount?: number; activeOperatorCount?: number; labels?: Partial<SettingsLabels> } }>();
	const i18n = $derived(createCurrentAdminI18n());
	const settings = useAdminSettings({
		readLocale: getCurrentAdminLocale,
		writeLocale: setCurrentAdminLocale,
	});
	const labels = $derived<SettingsLabels>({
		title: data?.labels?.title ?? i18n.t('settings.title'),
		description: data?.labels?.description ?? i18n.t('settings.description'),
		languageTitle: data?.labels?.languageTitle ?? i18n.t('settings.languageTitle'),
		languageDescription: data?.labels?.languageDescription ?? i18n.t('settings.languageDescription'),
		languageLabel: data?.labels?.languageLabel ?? i18n.t('settings.languageLabel'),
		languageJapanese: data?.labels?.languageJapanese ?? i18n.t('settings.languageJapanese'),
		languageEnglish: data?.labels?.languageEnglish ?? i18n.t('settings.languageEnglish'),
		languageSubmit: data?.labels?.languageSubmit ?? i18n.t('settings.languageSubmit'),
		languageSuccess: data?.labels?.languageSuccess ?? i18n.t('settings.languageSuccess'),
		languageError: data?.labels?.languageError ?? i18n.t('settings.languageError'),
		managementTitle: data?.labels?.managementTitle ?? i18n.t('settings.managementTitle'),
		managementDescription: data?.labels?.managementDescription ?? i18n.t('settings.managementDescription', { active: data?.activeOperatorCount ?? 0, total: data?.operatorCount ?? 0 }),
		managementBody: data?.labels?.managementBody ?? i18n.t('settings.managementBody'),
		managementButton: data?.labels?.managementButton ?? i18n.t('settings.managementButton'),
	});

	function saveLocale(): void {
		// locale 永続化の orchestration は domain action に委譲する。
		settings.actions.saveLocale();
	}
</script>

<svelte:head>
	<title>{labels.title}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">{labels.title}</h1>
		<p class="text-muted-foreground">{labels.description}</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{labels.languageTitle}</CardNS.CardTitle>
			<CardNS.CardDescription>{labels.languageDescription}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			{#if settings.data.state.localeUpdated}
				<p class="rounded-md border border-success/20 bg-success/10 p-3 text-sm text-success">{labels.languageSuccess}</p>
			{/if}
			{#if settings.data.state.localeError}
				<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{labels.languageError}</p>
			{/if}
			<div class="space-y-3">
				<div class="space-y-2">
					<Label for="operator-locale">{labels.languageLabel}</Label>
					<Select.Select type="single" bind:value={settings.data.state.selectedLocale}>
						<Select.SelectTrigger id="operator-locale">
							<Select.SelectValue>
								{settings.data.state.selectedLocale === 'ja' ? labels.languageJapanese : labels.languageEnglish}
							</Select.SelectValue>
						</Select.SelectTrigger>
						<Select.SelectContent>
							<Select.SelectItem value="ja">{labels.languageJapanese}</Select.SelectItem>
							<Select.SelectItem value="en">{labels.languageEnglish}</Select.SelectItem>
						</Select.SelectContent>
					</Select.Select>
				</div>
				<Button type="button" onclick={saveLocale}>{labels.languageSubmit}</Button>
			</div>
		</CardNS.CardContent>
	</CardNS.Card>
	{#if data?.canManageOperators ?? true}
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{labels.managementTitle}</CardNS.CardTitle>
			<CardNS.CardDescription>{labels.managementDescription}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<p class="text-sm text-muted-foreground">{labels.managementBody}</p>
			<Separator />
			<Button href="/settings/operators">{labels.managementButton}</Button>
		</CardNS.CardContent>
	</CardNS.Card>
	{/if}
</main>
