<script lang="ts">
	import { cn, type WithElementRef, type WithoutChildren } from "@www-template/ui/lib/utils";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		value,
		...restProps
	}: WithoutChildren<WithElementRef<HTMLAttributes<HTMLPreElement>>> & {
		value: string;
	} = $props();
</script>

<pre
	bind:this={ref}
	data-slot="code-block"
	class={cn("max-h-48 overflow-auto rounded bg-muted p-2 text-xs", className)}
	{...restProps}
>{value}</pre>
