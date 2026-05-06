<script lang="ts">
	import { Separator } from "@ui/components/separator";
	import { cn } from "@ui/lib/utils";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComponentProps<typeof Separator> = $props();
</script>

<Separator
	bind:ref
	data-slot="item-separator"
	orientation="horizontal"
	class={cn("my-2", className)}
	{...restProps}
/>
