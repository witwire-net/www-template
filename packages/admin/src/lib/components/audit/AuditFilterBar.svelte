<script lang="ts">
	import { Button, Field, Input, Label, Select } from '@www-template/ui/components';

	interface Operator { id: string; email: string; }

	const {
		onFilter,
		operators = [],
		labels,
	}: {
		onFilter?: (filters: Record<string, string | undefined>) => void;
		operators?: Operator[];
		labels: Record<string, string>;
	} = $props();

	let operatorId = $state('');
	let action = $state('');
	let dateFrom = $state('');
	let dateTo = $state('');

	const selectedOperatorLabel = $derived(
		operators.find((operator) => operator.id === operatorId)?.email ?? labels.all
	);

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
