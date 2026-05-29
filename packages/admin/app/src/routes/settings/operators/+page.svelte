<script lang="ts">
	import { createAdminOperator } from '@www-template/admin-domain';
	import { Button, CardNS, Dialog, Input, Label, Select, Spinner } from '@www-template/ui/components';

	import OperatorTable from '$lib/components/operators/OperatorTable.svelte';
	import { createCurrentAdminI18n } from '$lib/i18n';

	interface OperatorRow {
		id: string;
		email: string;
		displayName: string;
		role: string;
		isActive: boolean;
		lastLoginAt: Date | null;
	}

	interface OperatorsData { operators: OperatorRow[]; currentOperatorId: string }

	const { data, form } = $props<{
		data?: Partial<OperatorsData>;
		form?: { messageKey?: string };
	}>();
	let createdOperators = $state<OperatorRow[]>([]);
	const pageData = $derived<OperatorsData>({
		operators: [...createdOperators, ...(data?.operators ?? [])],
		currentOperatorId: data?.currentOperatorId ?? '',
	});
	const i18n = $derived(createCurrentAdminI18n());

	let addOpen = $state(false);
	let newOperatorEmail = $state('');
	let newOperatorRole = $state('viewer');
	let isCreating = $state(false);
	let createMessage = $state<string | null>(null);
	const tableOperators = $derived(pageData.operators.map((operator: OperatorRow) => ({ id: operator.id, email: operator.email, display_name: operator.displayName, role: operator.role, role_label: operatorRoleLabel(operator.role), is_active: operator.isActive, last_login_at: operator.lastLoginAt === null ? null : i18n.formatDateTime(operator.lastLoginAt) })));
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

	async function submitCreateOperator(): Promise<void> {
		// operator 作成は二重送信を止め、Admin API の CSRF/RBAC 検証へ一度だけ委譲する。
		if (isCreating) return;
		isCreating = true;
		createMessage = null;

		try {
			const result = await createAdminOperator({ email: newOperatorEmail, role: toOperatorRole(newOperatorRole) });
			if (!result.success) {
				createMessage = operatorErrorMessage(result.error);
				return;
			}

			// response に setup token 平文は含まれないため、作成済み summary だけを一覧に即時反映する。
			createdOperators = [
				{
					id: result.data.id,
					email: result.data.email,
					displayName: result.data.email,
					role: result.data.role,
					isActive: result.data.active,
					lastLoginAt: null,
				},
				...createdOperators,
			];
			newOperatorEmail = '';
			newOperatorRole = 'viewer';
			addOpen = false;
		} finally {
			// 成功・失敗のどちらでも form を再操作可能に戻す。
			isCreating = false;
		}
	}

	function toOperatorRole(role: string): 'admin' | 'operator' | 'viewer' {
		// Select から来る文字列を contract の role union に絞り、未知値は最小権限 viewer に落とす。
		if (role === 'admin' || role === 'operator') return role;
		return 'viewer';
	}

	function operatorRoleLabel(role: string): string {
		// role 表示は既存辞書 key を通し、未知 role だけ監査性のため raw 値を表示する。
		if (role === 'admin') return i18n.t('operators.roleAdmin');
		if (role === 'operator') return i18n.t('operators.roleOperator');
		if (role === 'viewer') return i18n.t('operators.roleViewer');
		return role;
	}

	function operatorErrorMessage(error: string): string {
		// domain error 分類だけを表示へ写像し、setup token delivery や権限判定の詳細は隠す。
		if (error === 'unauthenticated') return i18n.t('accounts.errorUnauthenticated');
		if (error === 'forbidden') return i18n.t('accounts.errorForbidden');
		if (error === 'duplicate-email') return i18n.t('accounts.errorDuplicate');
		if (error === 'unavailable') return i18n.t('accounts.errorUnavailable');
		return i18n.t('operators.createError');
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
				<div class="space-y-4">
					<div class="space-y-2"><Label for="email">{i18n.t('operators.email')}</Label><Input id="email" name="email" type="email" bind:value={newOperatorEmail} disabled={isCreating} required /></div>
					<div class="space-y-2"><Label for="role">{i18n.t('operators.role')}</Label><Select.Select name="role" type="single" bind:value={newOperatorRole}><Select.SelectTrigger id="role"><Select.SelectValue>{newOperatorRole === 'admin' ? i18n.t('operators.roleAdmin') : newOperatorRole === 'operator' ? i18n.t('operators.roleOperator') : i18n.t('operators.roleViewer')}</Select.SelectValue></Select.SelectTrigger><Select.SelectContent><Select.SelectItem value="admin">{i18n.t('operators.roleAdmin')}</Select.SelectItem><Select.SelectItem value="operator">{i18n.t('operators.roleOperator')}</Select.SelectItem><Select.SelectItem value="viewer">{i18n.t('operators.roleViewer')}</Select.SelectItem></Select.SelectContent></Select.Select></div>
					{#if createMessage !== null}<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{createMessage}</p>{/if}
					<Button type="button" disabled={isCreating || newOperatorEmail.trim() === ''} onclick={() => { void submitCreateOperator(); }}>
						{#if isCreating}<Spinner aria-hidden="true" />{/if}
						{i18n.t('operators.create')}
					</Button>
				</div>
			</Dialog.DialogContent>
		</Dialog.Dialog>
	</section>

	{#if form?.messageKey != null}<p class="rounded-md border border-error/20 bg-error/10 p-3 text-sm text-error">{i18n.t(form.messageKey)}</p>{/if}

	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>{i18n.t('operators.tableTitle')}</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<OperatorTable operators={tableOperators} labels={operatorTableLabels} currentOperatorId={pageData.currentOperatorId} />
		</CardNS.CardContent>
	</CardNS.Card>
</main>
