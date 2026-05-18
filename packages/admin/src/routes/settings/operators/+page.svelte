<script lang="ts">
	import { Button, CardNS, Dialog, Input, Label, Select, Separator } from '@www-template/ui/components';

	import OperatorTable from '$lib/components/operators/OperatorTable.svelte';

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
		data: { operators: OperatorRow[]; currentOperatorId: string; csrfToken: string };
		form?: { setupToken?: string; setupTokenEmail?: string; message?: string };
	}>();

	let addOpen = $state(false);
	let roleOpen = $state(false);
	let selectedOperatorId = $state('');
	let selectedRole = $state('viewer');
	const tableOperators = $derived(data.operators.map((operator: OperatorRow) => ({ id: operator.id, email: operator.email, display_name: operator.displayName, role: operator.role, is_active: operator.isActive, last_login_at: operator.lastLoginAt === null ? null : new Date(operator.lastLoginAt).toISOString() })));

	function editRole(id: string, role: string): void {
		// 現在値を dialog に転記し、明示的な保存操作まで変更しない。
		selectedOperatorId = id;
		selectedRole = role;
		roleOpen = true;
	}
</script>

<main class="space-y-6 p-8">
	<section class="flex items-end justify-between gap-4">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold tracking-tight">Operators</h1>
			<p class="text-slate-600">Admin Console へ入れるオペレーターと初回セットアップトークンを管理します。</p>
		</div>
		<Dialog.Dialog bind:open={addOpen}>
			<Dialog.DialogTrigger><Button>Add operator</Button></Dialog.DialogTrigger>
			<Dialog.DialogContent>
				<Dialog.DialogHeader><Dialog.DialogTitle>Add operator</Dialog.DialogTitle><Dialog.DialogDescription>登録後に一度だけ表示される setup token を共有します。</Dialog.DialogDescription></Dialog.DialogHeader>
				<form method="POST" action="?/create" class="space-y-4">
					<Input type="hidden" name="_csrf" value={data.csrfToken} />
					<div class="space-y-2"><Label for="email">Email</Label><Input id="email" name="email" type="email" required /></div>
					<div class="space-y-2"><Label for="displayName">Display name</Label><Input id="displayName" name="displayName" required /></div>
					<div class="space-y-2"><Label for="role">Role</Label><Select.Select name="role" type="single" value="viewer"><Select.SelectTrigger id="role"><Select.SelectValue>viewer</Select.SelectValue></Select.SelectTrigger><Select.SelectContent><Select.SelectItem value="admin">admin</Select.SelectItem><Select.SelectItem value="operator">operator</Select.SelectItem><Select.SelectItem value="viewer">viewer</Select.SelectItem></Select.SelectContent></Select.Select></div>
					<Button type="submit">Create</Button>
				</form>
			</Dialog.DialogContent>
		</Dialog.Dialog>
	</section>

	{#if form?.message != null}<p class="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{form.message}</p>{/if}
	{#if form?.setupToken != null}
		<CardNS.Card class="border-amber-200 bg-amber-50">
			<CardNS.CardHeader><CardNS.CardTitle>One-time setup token</CardNS.CardTitle><CardNS.CardDescription>{form.setupTokenEmail} にこの値を一度だけ安全に共有してください。</CardNS.CardDescription></CardNS.CardHeader>
			<CardNS.CardContent><code class="break-all rounded bg-white p-3 text-sm">{form.setupToken}</code></CardNS.CardContent>
		</CardNS.Card>
	{/if}

	<CardNS.Card>
		<CardNS.CardHeader><CardNS.CardTitle>Operator table</CardNS.CardTitle></CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<OperatorTable operators={tableOperators} currentOperatorId={data.currentOperatorId} onEditRole={editRole} />
			<Separator />
			<div class="grid gap-3 md:grid-cols-2">
				<form method="POST" action="?/deactivate" class="flex gap-2"><Input type="hidden" name="_csrf" value={data.csrfToken} /><Input name="operatorId" placeholder="operator id to deactivate" /><Button type="submit" variant="destructive">Deactivate</Button></form>
				<form method="POST" action="?/rotate" class="flex gap-2"><Input type="hidden" name="_csrf" value={data.csrfToken} /><Input name="operatorId" placeholder="operator id to rotate token" /><Button type="submit" variant="outline">Rotate setup token</Button></form>
			</div>
		</CardNS.CardContent>
	</CardNS.Card>

	<Dialog.Dialog bind:open={roleOpen}>
		<Dialog.DialogContent>
			<Dialog.DialogHeader><Dialog.DialogTitle>Update role</Dialog.DialogTitle></Dialog.DialogHeader>
			<form method="POST" action="?/update" class="space-y-4">
				<Input type="hidden" name="_csrf" value={data.csrfToken} />
				<Input name="operatorId" bind:value={selectedOperatorId} readonly />
				<Select.Select name="role" type="single" value={selectedRole} onValueChange={(next: string) => { selectedRole = next; }}><Select.SelectTrigger><Select.SelectValue>{selectedRole}</Select.SelectValue></Select.SelectTrigger><Select.SelectContent><Select.SelectItem value="admin">admin</Select.SelectItem><Select.SelectItem value="operator">operator</Select.SelectItem><Select.SelectItem value="viewer">viewer</Select.SelectItem></Select.SelectContent></Select.Select>
				<Button type="submit">Save role</Button>
			</form>
		</Dialog.DialogContent>
	</Dialog.Dialog>
</main>
