<script lang="ts">
	import { Badge } from '@www-template/ui/components';

	const { role, label = role }: { role: string; label?: string } = $props();

	const variant = $derived(
		role === 'admin' ? 'default' : role === 'operator' ? 'info' : 'secondary'
	);
</script>

<Badge {variant}>{label}</Badge>
