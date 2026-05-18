<script lang="ts">
	import { Badge } from '@www-template/ui/components';

	const { status }: { status: string } = $props();

	const variant = $derived(status === 'active' ? 'success' : 'danger');
</script>

<Badge {variant}>{status}</Badge>
