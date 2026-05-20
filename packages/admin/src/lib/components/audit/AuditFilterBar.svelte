<script lang="ts">
	import { Button, Field, Input, Label, Select } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	interface Operator { id: string; email: string; }

	const {
		onFilter,
		operators = [],
		labels = createAuditFilterLabels(),
	}: {
		onFilter?: (filters: Record<string, string | undefined>) => void;
		operators?: Operator[];
		labels?: ReturnType<typeof createAuditFilterLabels>;
	} = $props();

	let operatorId = $state('');
	let action = $state('');
	let dateFrom = $state('');
	let dateTo = $state('');

	const selectedOperatorLabel = $derived(
		operators.find((operator) => operator.id === operatorId)?.email ?? labels.all
	);

	function createAuditFilterLabels() {
		// 監査 filter component 単体利用時も Admin-owned fallback 辞書に閉じる。
		const { t } = createAdminI18n();
		return {
			operator: t('audit.operator'),
			all: t('audit.all'),
			action: t('audit.action'),
			actionPlaceholder: t('audit.actionPlaceholder'),
			from: t('audit.from'),
			to: t('audit.to'),
			filter: t('audit.filter'),
			clear: t('audit.clear'),
		};
	}

	function handleFilter(): void {
		onFilter?.({
			operatorId: operatorId !== '' ? operatorId : undefined,
			action: action !== '' ? action : undefined,
			dateFrom: dateFrom !== '' ? dateFrom : undefined,
			dateTo: dateTo !== '' ? dateTo : undefined,
		});
	}

	function handleClear(): void {
		operatorId = '';
		action = '';
		dateFrom = '';
		dateTo = '';
		onFilter?.({});
	}
</script>

<Field.FieldGroup>
	<Field.Field>
		<Label for="filter-operator">{labels.operator}</Label>
		<Select.Select type="single" value={operatorId} onValueChange={(v: string) => { operatorId = v; }}>
			<Select.SelectTrigger id="filter-operator">
				<Select.SelectValue>{selectedOperatorLabel}</Select.SelectValue>
			</Select.SelectTrigger>
			<Select.SelectContent>
				<Select.SelectItem value="">{labels.all}</Select.SelectItem>
				{#each operators as op (op.id)}
					<Select.SelectItem value={op.id}>{op.email}</Select.SelectItem>
				{/each}
			</Select.SelectContent>
		</Select.Select>
	</Field.Field>
	<Field.Field>
		<Label for="filter-action">{labels.action}</Label>
		<Input id="filter-action" placeholder={labels.actionPlaceholder} bind:value={action} />
	</Field.Field>
	<Field.Field>
		<Label for="filter-from">{labels.from}</Label>
		<Input id="filter-from" type="date" bind:value={dateFrom} />
	</Field.Field>
	<Field.Field>
		<Label for="filter-to">{labels.to}</Label>
		<Input id="filter-to" type="date" bind:value={dateTo} />
	</Field.Field>
	<Button onclick={handleFilter}>{labels.filter}</Button>
	<Button variant="outline" onclick={handleClear}>{labels.clear}</Button>
</Field.FieldGroup>
