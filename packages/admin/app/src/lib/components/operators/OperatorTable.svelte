<script lang="ts">
	import { Badge, DropdownMenu, Table } from '@www-template/ui/components';

	import PaginationFooter from '$lib/components/shared/PaginationFooter.svelte';

	import OperatorRoleBadge from './OperatorRoleBadge.svelte';

	interface Operator {
		id: string;
		email: string;
		display_name: string;
		role: string;
		role_label?: string;
		is_active: boolean;
		last_login_at: string | null;
	}

	const {
		operators,
		currentOperatorId = '',
		onEditRole,
		onDeactivate,
		onRotateToken,
		page = 1,
		totalPages = 1,
		onPageChange,
		labels,
	}: {
		operators: Operator[];
		currentOperatorId?: string;
		onEditRole?: (id: string, role: string) => void;
		onDeactivate?: (id: string) => void;
		onRotateToken?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels: Record<string, string>;
	} = $props();

	const hasActions = $derived(onEditRole != null || onDeactivate != null || onRotateToken != null);
</script>

<Table.Table>
	<Table.TableCaption>{labels.caption}</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>{labels.email}</Table.TableHead>
			<Table.TableHead>{labels.displayName}</Table.TableHead>
			<Table.TableHead>{labels.role}</Table.TableHead>
			<Table.TableHead>{labels.status}</Table.TableHead>
			<Table.TableHead>{labels.lastLogin}</Table.TableHead>
			{#if hasActions}<Table.TableHead>{labels.actions}</Table.TableHead>{/if}
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#each operators as op (op.id)}
			<Table.TableRow>
				<Table.TableCell>{op.email}</Table.TableCell>
				<Table.TableCell>{op.display_name}</Table.TableCell>
				<Table.TableCell><OperatorRoleBadge role={op.role} label={op.role_label} /></Table.TableCell>
				<Table.TableCell>
					<Badge variant={op.is_active ? 'success' : 'secondary'}>
						{op.is_active ? labels.active : labels.inactive}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>{op.last_login_at ?? '-'}</Table.TableCell>
				{#if hasActions}
					<Table.TableCell>
						<DropdownMenu.DropdownMenu>
							<DropdownMenu.DropdownMenuTriggerButton variant="ghost" size="xs" aria-label={labels.manage + ' ' + op.email}>{labels.actions}</DropdownMenu.DropdownMenuTriggerButton>
							<DropdownMenu.DropdownMenuContent align="end">
								<DropdownMenu.DropdownMenuLabel>{labels.manage}</DropdownMenu.DropdownMenuLabel>
								<DropdownMenu.DropdownMenuSeparator />
								{#if onEditRole}
									<DropdownMenu.DropdownMenuItem onclick={() => { onEditRole(op.id, op.role); }}>
										{labels.editRole}
									</DropdownMenu.DropdownMenuItem>
								{/if}
								{#if onDeactivate != null && op.id !== currentOperatorId && op.is_active}
									<DropdownMenu.DropdownMenuItem onclick={() => { onDeactivate(op.id); }}>
										{labels.deactivate}
									</DropdownMenu.DropdownMenuItem>
								{/if}
								{#if onRotateToken != null}
									<DropdownMenu.DropdownMenuItem onclick={() => { onRotateToken(op.id); }}>
										{labels.rotate}
									</DropdownMenu.DropdownMenuItem>
								{/if}
							</DropdownMenu.DropdownMenuContent>
						</DropdownMenu.DropdownMenu>
					</Table.TableCell>
				{/if}
			</Table.TableRow>
		{/each}
	</Table.TableBody>
</Table.Table>

<PaginationFooter {page} {totalPages} perPage={10} {onPageChange} labels={labels} />
