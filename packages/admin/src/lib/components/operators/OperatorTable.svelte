<script lang="ts">
	import { Badge, DropdownMenu, Pagination, Table } from '@www-template/ui/components';

	import { createAdminI18n } from '$lib/i18n';

	import OperatorRoleBadge from './OperatorRoleBadge.svelte';

	interface Operator {
		id: string;
		email: string;
		display_name: string;
		role: string;
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
		labels = createOperatorTableLabels(),
	}: {
		operators: Operator[];
		currentOperatorId?: string;
		onEditRole?: (id: string, role: string) => void;
		onDeactivate?: (id: string) => void;
		onRotateToken?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
		labels?: ReturnType<typeof createOperatorTableLabels>;
	} = $props();

	function createOperatorTableLabels() {
		// オペレーター table component 単体利用時も fallback 辞書に閉じる。
		const { t } = createAdminI18n();
		return {
			caption: t('operators.tableCaption'),
			email: t('operators.email'),
			displayName: t('operators.displayName'),
			role: t('operators.role'),
			status: t('operators.status'),
			lastLogin: t('operators.lastLogin'),
			actions: t('operators.actions'),
			active: t('operators.active'),
			inactive: t('operators.inactive'),
			manage: t('operators.manage'),
			editRole: t('operators.editRole'),
			deactivate: t('operators.deactivate'),
			rotate: t('operators.rotate'),
			pagination: t('shared.pagination'),
			previousPage: t('shared.previousPage'),
			nextPage: t('shared.nextPage'),
		};
	}
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
			<Table.TableHead>{labels.actions}</Table.TableHead>
		</Table.TableRow>
	</Table.TableHeader>
	<Table.TableBody>
		{#each operators as op (op.id)}
			<Table.TableRow>
				<Table.TableCell>{op.email}</Table.TableCell>
				<Table.TableCell>{op.display_name}</Table.TableCell>
				<Table.TableCell><OperatorRoleBadge role={op.role} /></Table.TableCell>
				<Table.TableCell>
					<Badge variant={op.is_active ? 'success' : 'secondary'}>
						{op.is_active ? labels.active : labels.inactive}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>{op.last_login_at ?? '-'}</Table.TableCell>
				<Table.TableCell>
					<DropdownMenu.DropdownMenu>
						<DropdownMenu.DropdownMenuTriggerButton variant="ghost" size="xs">{labels.actions}</DropdownMenu.DropdownMenuTriggerButton>
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
			</Table.TableRow>
		{/each}
	</Table.TableBody>
</Table.Table>

{#if totalPages > 1}
	<Pagination.Pagination aria-label={labels.pagination} count={totalPages * 10} perPage={10} {page} onPageChange={(p: number) => { onPageChange?.(p); }}>
		<Pagination.PaginationContent>
			<Pagination.PaginationItem>
				<Pagination.PaginationPrevButton aria-label={labels.previousPage} />
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationLink page={{ type: 'page', value: page }} isActive>
					{page}
				</Pagination.PaginationLink>
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationNextButton aria-label={labels.nextPage} />
			</Pagination.PaginationItem>
		</Pagination.PaginationContent>
	</Pagination.Pagination>
{/if}
