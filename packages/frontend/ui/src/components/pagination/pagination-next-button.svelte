<script lang="ts">
	import { Pagination as PaginationPrimitive } from "bits-ui";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import { buttonVariants } from "@www-template/ui/components/button";
	import { cn } from "@www-template/ui/lib/utils";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: PaginationPrimitive.NextButtonProps = $props();
</script>

{#snippet Fallback()}
	<ChevronRightIcon class="size-4" />
{/snippet}

<PaginationPrimitive.NextButton
	bind:ref
	class={cn(buttonVariants({ variant: "ghost" }), "pr-2!", className)}
	{...restProps}
>
	{#if children}
		{@render children?.()}
	{:else}
		{@render Fallback()}
	{/if}
</PaginationPrimitive.NextButton>
