<script lang="ts">
	import { Pagination as PaginationPrimitive } from "bits-ui";
	import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
	import { buttonVariants } from "@www-template/ui/components/button";
	import { cn } from "@www-template/ui/lib/utils";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: PaginationPrimitive.PrevButtonProps = $props();
</script>

{#snippet Fallback()}
	<ChevronLeftIcon class="size-4" />
	<span>Previous</span>
{/snippet}

<PaginationPrimitive.PrevButton
	bind:ref
	aria-label="Go to previous page"
	class={cn(buttonVariants({ variant: "ghost" }), "pl-2!", className)}
	{...restProps}
>
	{#if children}
		{@render children?.()}
	{:else}
		{@render Fallback()}
	{/if}
</PaginationPrimitive.PrevButton>
