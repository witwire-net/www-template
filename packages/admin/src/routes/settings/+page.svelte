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
</script>

<svelte:head>
	<title>{data.labels.title} - Admin Console</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">{data.labels.title}</h1>
		<p class="text-slate-600">{data.labels.description}</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>{data.labels.languageTitle}</CardNS.CardTitle>
			<CardNS.CardDescription>{data.labels.languageDescription}</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			{#if data.localeUpdated}
				<p class="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">{data.labels.languageSuccess}</p>
			{/if}
			{#if form?.localeError === true}
				<p class="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{data.labels.languageError}</p>
			{/if}
			<form method="POST" action="?/locale" class="space-y-3">
				<Input type="hidden" name="_csrf" value={data.csrfToken} />
				<div class="space-y-2">
					<Label for="operator-locale">{data.labels.languageLabel}</Label>
					<Select.Select name="locale" type="single" value={data.locale}>
						<Select.SelectTrigger id="operator-locale"><Select.SelectValue>{data.locale === 'ja' ? data.labels.languageJapanese : data.labels.languageEnglish}</Select.SelectValue></Select.SelectTrigger>
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
			<p class="text-sm text-slate-600">{data.labels.managementBody}</p>
			<Separator />
			<Button href="/settings/operators">{data.labels.managementButton}</Button>
		</CardNS.CardContent>
	</CardNS.Card>
	{/if}
</main>
