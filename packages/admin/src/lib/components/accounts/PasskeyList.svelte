<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, Badge, EmptyState, Button, Item } from '@www-template/ui/components';

	interface PasskeyInfo {
		id: string;
		credential_handle: string;
		created_at: string;
	}

	const {
		passkeys,
		onAdd,
		onDelete,
	}: {
		passkeys: PasskeyInfo[];
		onAdd?: () => void;
		onDelete?: (id: string) => void;
	} = $props();

	function truncateHandle(handle: string): string {
		return handle.length > 20 ? handle.slice(0, 20) + '...' : handle;
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>Passkeys</CardTitle>
	</CardHeader>
	<CardContent>
		{#if passkeys.length === 0}
			<EmptyState title="No Passkeys" description="This account has no registered passkeys." />
		{:else}
			{#each passkeys as pk (pk.id)}
				<Item.Item>
					<Item.ItemContent>
						<Item.ItemTitle>{truncateHandle(pk.credential_handle)}</Item.ItemTitle>
						<Item.ItemDescription>{pk.created_at}</Item.ItemDescription>
					</Item.ItemContent>
					<Item.ItemActions>
						<Badge variant="outline">Passkey</Badge>
					{#if onDelete != null && passkeys.length > 1}
						<Button variant="destructive" size="xs" onclick={() => { onDelete(pk.id); }}>Delete</Button>
					{/if}
					</Item.ItemActions>
				</Item.Item>
			{/each}
		{/if}
		{#if onAdd}
			<Button onclick={onAdd}>Add Passkey</Button>
		{/if}
	</CardContent>
</Card>
