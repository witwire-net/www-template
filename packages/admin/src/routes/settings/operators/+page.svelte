<script lang="ts">
	import { Button, CardNS, Dialog, Input, Label, Select, Separator } from '@www-template/ui/components';

	import OperatorTable from '$lib/components/operators/OperatorTable.svelte';
	import { createAdminI18n } from '$lib/i18n';

	interface OperatorRow {
		id: string;
		email: string;
		displayName: string;
		role: string;
		isActive: boolean;
		lastLoginAt: Date | null;
		setupTokenHash: string | null;
	}

	const { data, form } = $props<{
		data: { locale: 'ja' | 'en'; operators: OperatorRow[]; currentOperatorId: string; csrfToken: string };
		form?: { setupToken?: string; setupTokenEmail?: string; messageKey?: string };
	}>();
	const i18n = $derived(createAdminI18n(data.locale));

	let addOpen = $state(false);
	let roleOpen = $state(false);
	let selectedOperatorId = $state('');
	let newOperatorRole = $state('viewer');
	let selectedRole = $state('viewer');
	const tableOperators = $derived(data.operators.map((operator: OperatorRow) => ({ id: operator.id, email: operator.email, display_name: operator.displayName, role: operator.role, is_active: operator.isActive, last_login_at: operator.lastLoginAt === null ? null : new Date(operator.lastLoginAt).toISOString() })));
	const operatorTableLabels = $derived({
		caption: i18n.t('operators.tableCaption'),
		email: i18n.t('operators.email'),
		displayName: i18n.t('operators.displayName'),
		role: i18n.t('operators.role'),
		status: i18n.t('operators.status'),
		lastLogin: i18n.t('operators.lastLogin'),
		actions: i18n.t('operators.actions'),
		active: i18n.t('operators.active'),
		inactive: i18n.t('operators.inactive'),
		manage: i18n.t('operators.manage'),
		editRole: i18n.t('operators.editRole'),
		deactivate: i18n.t('operators.deactivate'),
		rotate: i18n.t('operators.rotate'),
		pagination: i18n.t('shared.pagination'),
		previousPage: i18n.t('shared.previousPage'),
		nextPage: i18n.t('shared.nextPage'),
	});

	function editRole(id: string, role: string): void {
		// 現在値を dialog に転記し、明示的な保存操作まで変更しない。
		selectedOperatorId = id;
		selectedRole = role;
		roleOpen = true;
	}
</script>

<svelte:head>
	<title>{i18n.t('operators.title')}</title>
</svelte:head>

<main class="space-y-6 p-8">
	<section class="flex items-end justify-between gap-4">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold tracking-tight">{i18n.t('operators.title')}</h1>
			<p class="text-muted-foreground">{i18n.t('operators.description')}</p>
		</div>
		<Dialog.Dialog bind:open={addOpen}>
			<Dialog.DialogTrigger><Button>{i18n.t('operators.add')}</Button></Dialog.DialogTrigger>
			<Dialog.DialogContent closeLabel={i18n.t('shared.close')}>
				<Dialog.DialogHeader><Dialog.DialogTitle>{i18n.t('operators.add')}</Dialog.DialogTitle><Dialog.DialogDescription>{i18n.t('operators.addDescription')}</Dialog.DialogDescription></Dialog.DialogHeader>
				<form method="POST" action="?/create" class="space-y-4">
					<Input type="hidden" name="_csrf" value={data.csrfToken} />
					<div class="space-y-2"><Label for="email">{i18n.t('operators.email')}</Label><Input id="email" name="email" type="email" required /></div>
					<div class="space-y-2"><Label for="displayName">{i18n.t('operators.displayName')}</Label><Input id="displayName" name="displayName" required /></div>
					<div class="space-y-2"><Label for="role">{i18n.t('operators.role')}</Label><Select.Select name="role" type="single" bind:value={newOperatorRole}><Select.SelectTrigger id="role"><Select.SelectValue>{newOperatorRole === 'admin' ? i18n.t('operators.roleAdmin') : newOperatorRole === 'operator' ? i18n.t('operators.roleOperator') : i18n.t('operators.roleViewer')}</Select.SelectValue></Select.SelectTrigger><Select.SelectContent><Select.SelectItem value="admin">{i18n.t('operators.roleAdmin')}</Select.SelectItem><Select.SelectItem value="operator">{i18n.t('operators.roleOperator')}</Select.SelectItem><Select.SelectItem value="viewer">{i18n.t('operators.roleViewer')}</Select.SelectItem></Select.SelectContent></Select.Select></div>
					<Button type="submit">{i18n.t('operators.create')}</Button>
				</form>
			</Dialog.DialogContent>
		</Dialog.Dialog>
	</section>

	{#if form?.messageKey != null}<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{i18n.t(form.messageKey)}</p>{/if}
	{#if form?.setupToken != null}
		<CardNS.Card class="border-warning/20 bg-warning/10">
			<CardNS.CardHeader><CardNS.CardTitle>{i18n.t('operators.setupTokenTitle')}</CardNS.CardTitle><CardNS.CardDescription>{i18n.t('operators.setupTokenDescription', { email: form.setupTokenEmail ?? '' })}</CardNS.CardDescription></CardNS.CardHeader>
			<CardNS.CardContent><code class="break-all rounded bg-surface p-3 text-sm">{form.setupToken}</code></CardNS.CardContent>
		</CardNS.Card>
	{/if}

	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>{i18n.t('operators.tableTitle')}</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<OperatorTable operators={tableOperators} labels={operatorTableLabels} currentOperatorId={data.currentOperatorId} onEditRole={editRole} />
			<Separator />
			<div class="grid gap-3 md:grid-cols-2">
				<form method="POST" action="?/deactivate" class="flex gap-2"><Input type="hidden" name="_csrf" value={data.csrfToken} /><Input name="operatorId" placeholder={i18n.t('operators.deactivatePlaceholder')} /><Button type="submit" variant="destructive">{i18n.t('operators.deactivate')}</Button></form>
				<form method="POST" action="?/rotate" class="flex gap-2"><Input type="hidden" name="_csrf" value={data.csrfToken} /><Input name="operatorId" placeholder={i18n.t('operators.rotatePlaceholder')} /><Button type="submit" variant="outline">{i18n.t('operators.rotate')}</Button></form>
			</div>
		</CardNS.CardContent>
	</CardNS.Card>

	<Dialog.Dialog bind:open={roleOpen}>
		<Dialog.DialogContent closeLabel={i18n.t('shared.close')}>
			<Dialog.DialogHeader><Dialog.DialogTitle>{i18n.t('operators.updateRole')}</Dialog.DialogTitle></Dialog.DialogHeader>
			<form method="POST" action="?/update" class="space-y-4">
				<Input type="hidden" name="_csrf" value={data.csrfToken} />
				<Input name="operatorId" bind:value={selectedOperatorId} readonly />
				<Select.Select name="role" type="single" value={selectedRole} onValueChange={(next: string) => { selectedRole = next; }}><Select.SelectTrigger><Select.SelectValue>{selectedRole}</Select.SelectValue></Select.SelectTrigger><Select.SelectContent><Select.SelectItem value="admin">{i18n.t('operators.roleAdmin')}</Select.SelectItem><Select.SelectItem value="operator">{i18n.t('operators.roleOperator')}</Select.SelectItem><Select.SelectItem value="viewer">{i18n.t('operators.roleViewer')}</Select.SelectItem></Select.SelectContent></Select.Select>
				<Button type="submit">{i18n.t('operators.saveRole')}</Button>
			</form>
		</Dialog.DialogContent>
	</Dialog.Dialog>
</main>
