<script lang="ts">
	import { Separator as SeparatorPrimitive } from "bits-ui";
	import { cn } from "@www-template/ui/lib/utils";

	let {
		ref = $bindable(null),
		class: className,
		"data-slot": dataSlot = "separator",
		...restProps
	}: SeparatorPrimitive.RootProps = $props();
</script>

<SeparatorPrimitive.Root
	bind:ref
	data-slot={dataSlot}
	class={cn(
		"shrink-0 data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-px data-[orientation=horizontal]:bg-gradient-to-r data-[orientation=horizontal]:from-transparent data-[orientation=horizontal]:via-brand-500 data-[orientation=horizontal]:to-transparent data-[orientation=vertical]:h-full data-[orientation=vertical]:bg-border",
		className
	)}
	{...restProps}
/>
