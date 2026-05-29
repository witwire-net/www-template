<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, Badge, EmptyState, Button, Item } from '@www-template/ui/components';

	import { createCurrentAdminI18n } from '$lib/i18n';

	interface PasskeyInfo {
		id: string;
		credential_handle: string;
		created_at: string;
	}

	const {
		passkeys,
		onAdd,
		onDelete,
		labels = createPasskeyListLabels(),
	}: {
		passkeys: PasskeyInfo[];
		onAdd?: () => void;
		onDelete?: (id: string) => void;
		labels?: ReturnType<typeof createPasskeyListLabels>;
	} = $props();

	function createPasskeyListLabels() {
		// Account detail 以外から使われた場合も現在の Admin locale で表示できるようにする。
		const { t } = createCurrentAdminI18n();
		return {
			title: t('passkeyList.title'),
			emptyTitle: t('passkeyList.emptyTitle'),
			emptyDescription: t('passkeyList.emptyDescription'),
			badge: t('passkeyList.badge'),
			delete: t('passkeyList.delete'),
			add: t('passkeyList.add'),
		};
	}

	function truncateHandle(handle: string): string {
		return handle.length > 20 ? handle.slice(0, 20) + '...' : handle;
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>{labels.title}</CardTitle>
	</CardHeader>
	<CardContent>
		{#if passkeys.length === 0}
			<EmptyState title={labels.emptyTitle} description={labels.emptyDescription} />
		{:else}
			{#each passkeys as pk (pk.id)}
				<Item.Item>
					<Item.ItemContent>
						<Item.ItemTitle>{truncateHandle(pk.credential_handle)}</Item.ItemTitle>
						<Item.ItemDescription>{pk.created_at}</Item.ItemDescription>
					</Item.ItemContent>
					<Item.ItemActions>
						<Badge variant="outline">{labels.badge}</Badge>
					{#if onDelete != null && passkeys.length > 1}
						<Button variant="destructive" size="xs" onclick={() => { onDelete(pk.id); }}>{labels.delete}</Button>
					{/if}
					</Item.ItemActions>
				</Item.Item>
			{/each}
		{/if}
		{#if onAdd}
			<Button onclick={onAdd}>{labels.add}</Button>
		{/if}
	</CardContent>
</Card>
