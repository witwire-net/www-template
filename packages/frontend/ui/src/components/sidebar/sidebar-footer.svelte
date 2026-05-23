<script lang="ts">
	import type { HTMLAttributes } from "svelte/elements";
	import { cn, type WithElementRef } from "@www-template/ui/lib/utils";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="sidebar-footer"
	data-sidebar="footer"
	class={cn("gap-2 p-4 flex flex-col", className)}
	{...restProps}
>
	{@render children?.()}
</div>
