<script lang="ts">
	import { Button, CardNS, Input, Label, Select, Separator } from '@www-template/ui/components';

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

	const { data, form } = $props<{ data: { locale: 'ja' | 'en'; localeUpdated: boolean; canManageOperators: boolean; operatorCount: number; activeOperatorCount: number; labels: SettingsLabels; csrfToken: string }; form?: { localeError?: boolean } }>();

	let selectedLocale = $state<'ja' | 'en'>('ja');

	$effect.pre(() => {
		selectedLocale = data.locale;
	});
</script>

<svelte:head>
	<title>{data.labels.title}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">{data.labels.title}</h1>
		<p class="text-muted-foreground">{data.labels.description}</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{data.labels.languageTitle}</CardNS.CardTitle>
			<CardNS.CardDescription>{data.labels.languageDescription}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			{#if data.localeUpdated}
				<p class="rounded-md border border-success/20 bg-success/10 p-3 text-sm text-success">{data.labels.languageSuccess}</p>
			{/if}
			{#if form?.localeError === true}
				<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{data.labels.languageError}</p>
			{/if}
			<form method="POST" action="?/locale" class="space-y-3">
				<Input type="hidden" name="_csrf" value={data.csrfToken} />
				<Input type="hidden" name="locale" value={selectedLocale} />
				<div class="space-y-2">
					<Label for="operator-locale">{data.labels.languageLabel}</Label>
					<Select.Select type="single" bind:value={selectedLocale}>
						<Select.SelectTrigger id="operator-locale">
							<Select.SelectValue>
								{selectedLocale === 'ja' ? data.labels.languageJapanese : data.labels.languageEnglish}
							</Select.SelectValue>
						</Select.SelectTrigger>
						<Select.SelectContent>
							<Select.SelectItem value="ja">{data.labels.languageJapanese}</Select.SelectItem>
							<Select.SelectItem value="en">{data.labels.languageEnglish}</Select.SelectItem>
						</Select.SelectContent>
					</Select.Select>
				</div>
				<Button type="submit">{data.labels.languageSubmit}</Button>
			</form>
		</CardNS.CardContent>
	</CardNS.Card>
	{#if data.canManageOperators}
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{data.labels.managementTitle}</CardNS.CardTitle>
			<CardNS.CardDescription>{data.labels.managementDescription}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<p class="text-sm text-muted-foreground">{data.labels.managementBody}</p>
			<Separator />
			<Button href="/settings/operators">{data.labels.managementButton}</Button>
		</CardNS.CardContent>
	</CardNS.Card>
	{/if}
</main>
