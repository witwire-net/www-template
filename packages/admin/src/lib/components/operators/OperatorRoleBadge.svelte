<script lang="ts">
	import { Badge } from '@www-template/ui/components';

	const { role }: { role: string } = $props();

	const variant = $derived(
		role === 'admin' ? 'default' : role === 'operator' ? 'info' : 'secondary'
	);
</script>

<Badge {variant}>{role}</Badge>
