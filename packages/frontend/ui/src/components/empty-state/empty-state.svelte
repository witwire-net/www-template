<script lang="ts">
	import {
		Empty,
		EmptyTitle,
		EmptyDescription,
	} from "@www-template/ui/components/empty";

	interface Props {
		title?: string;
		description?: string;
		class?: string;
	}

	let {
		title,
		description,
		class: className,
	}: Props = $props();
</script>

<Empty class={className}>
	{#if title}
		<EmptyTitle>{title}</EmptyTitle>
	{/if}
	{#if description}
		<EmptyDescription>{description}</EmptyDescription>
	{/if}
</Empty>
