<script lang="ts">
	import type { Snippet } from "svelte";
	import { cn } from "@www-template/ui/lib/utils";

	interface Props {
		placeholder?: string;
		children?: Snippet;
		class?: string;
	}

	let { placeholder, children, class: className }: Props = $props();
</script>

<span
	data-slot="select-value"
	class={cn("flex items-center gap-1.5 line-clamp-1", className)}
>
	{#if children}
		{@render children()}
	{:else}
		{placeholder ?? ''}
	{/if}
</span>
