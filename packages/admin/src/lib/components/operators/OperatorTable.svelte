<script lang="ts">
	import { Badge, DropdownMenu, Pagination, Table } from '@www-template/ui/components';

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
	}: {
		operators: Operator[];
		currentOperatorId?: string;
		onEditRole?: (id: string, role: string) => void;
		onDeactivate?: (id: string) => void;
		onRotateToken?: (id: string) => void;
		page?: number;
		totalPages?: number;
		onPageChange?: (page: number) => void;
	} = $props();
</script>

<Table.Table>
	<Table.TableCaption>Operator list</Table.TableCaption>
	<Table.TableHeader>
		<Table.TableRow>
			<Table.TableHead>Email</Table.TableHead>
			<Table.TableHead>Display Name</Table.TableHead>
			<Table.TableHead>Role</Table.TableHead>
			<Table.TableHead>Status</Table.TableHead>
			<Table.TableHead>Last Login</Table.TableHead>
			<Table.TableHead>Actions</Table.TableHead>
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
						{op.is_active ? 'Active' : 'Inactive'}
					</Badge>
				</Table.TableCell>
				<Table.TableCell>{op.last_login_at ?? '-'}</Table.TableCell>
				<Table.TableCell>
					<DropdownMenu.DropdownMenu>
						<DropdownMenu.DropdownMenuTriggerButton variant="ghost" size="xs">Actions</DropdownMenu.DropdownMenuTriggerButton>
						<DropdownMenu.DropdownMenuContent align="end">
							<DropdownMenu.DropdownMenuLabel>Manage</DropdownMenu.DropdownMenuLabel>
							<DropdownMenu.DropdownMenuSeparator />
							{#if onEditRole}
								<DropdownMenu.DropdownMenuItem onclick={() => { onEditRole(op.id, op.role); }}>
									Edit Role
								</DropdownMenu.DropdownMenuItem>
							{/if}
							{#if onDeactivate != null && op.id !== currentOperatorId && op.is_active}
								<DropdownMenu.DropdownMenuItem onclick={() => { onDeactivate(op.id); }}>
									Deactivate
								</DropdownMenu.DropdownMenuItem>
							{/if}
							{#if onRotateToken != null}
								<DropdownMenu.DropdownMenuItem onclick={() => { onRotateToken(op.id); }}>
									Rotate Setup Token
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
	<Pagination.Pagination count={totalPages * 10} perPage={10} {page} onPageChange={(p: number) => { onPageChange?.(p); }}>
		<Pagination.PaginationContent>
			<Pagination.PaginationItem>
				<Pagination.PaginationPrevButton />
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationLink page={{ type: 'page', value: page }} isActive>
					{page}
				</Pagination.PaginationLink>
			</Pagination.PaginationItem>
			<Pagination.PaginationItem>
				<Pagination.PaginationNextButton />
			</Pagination.PaginationItem>
		</Pagination.PaginationContent>
	</Pagination.Pagination>
{/if}
