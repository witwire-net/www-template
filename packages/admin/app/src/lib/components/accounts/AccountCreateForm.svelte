<script lang="ts">
	import { Button, CardNS, Field, Input, Label, Select, Spinner } from '@www-template/ui/components';

	interface AccountCreateFormLabels {
		title: string;
		description: string;
		email: string;
		locale: string;
		localeJa: string;
		localeEn: string;
		submit: string;
		submitting: string;
	}

	let {
		email = $bindable(''),
		locale = $bindable('ja'),
		message = null,
		isSubmitting = false,
		labels,
		onSubmit,
	}: {
		email?: string;
		locale?: 'ja' | 'en';
		message?: string | null;
		isSubmitting?: boolean;
		labels: AccountCreateFormLabels;
		onSubmit: () => void;
	} = $props();
</script>

<CardNS.Card>
	<CardNS.CardHeader>
		<CardNS.CardTitle>{labels.title}</CardNS.CardTitle>
		<CardNS.CardDescription>{labels.description}</CardNS.CardDescription>
	</CardNS.CardHeader>
	<CardNS.CardContent class="space-y-4">
		<Field.FieldGroup class="grid gap-4 md:grid-cols-3">
			<Field.Field>
				<Label for="account-create-email">{labels.email}</Label>
				<Input id="account-create-email" type="email" autocomplete="email" placeholder="customer@example.com" bind:value={email} disabled={isSubmitting} />
			</Field.Field>
			<Field.Field>
				<Label for="account-create-locale">{labels.locale}</Label>
				<Select.Select type="single" value={locale} onValueChange={(next: string) => { locale = next === 'en' ? 'en' : 'ja'; }}>
					<Select.SelectTrigger id="account-create-locale"><Select.SelectValue>{locale === 'ja' ? labels.localeJa : labels.localeEn}</Select.SelectValue></Select.SelectTrigger>
					<Select.SelectContent>
						<Select.SelectItem value="ja">{labels.localeJa}</Select.SelectItem>
						<Select.SelectItem value="en">{labels.localeEn}</Select.SelectItem>
					</Select.SelectContent>
				</Select.Select>
			</Field.Field>
			<Field.Field class="justify-end">
				<Button disabled={isSubmitting || email.trim() === ''} onclick={onSubmit}>
					{#if isSubmitting}
						<Spinner aria-hidden="true" />
						{labels.submitting}
					{:else}
						{labels.submit}
					{/if}
				</Button>
			</Field.Field>
		</Field.FieldGroup>
		{#if message !== null}
			<p class="rounded-2xl border border-destructive px-4 py-3 text-sm text-destructive" role="alert">{message}</p>
		{/if}
	</CardNS.CardContent>
</CardNS.Card>
