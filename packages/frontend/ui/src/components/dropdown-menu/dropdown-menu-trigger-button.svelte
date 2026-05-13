<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import { buttonVariants, type ButtonVariant, type ButtonSize } from "@www-template/ui/components/button";
	import { cn } from "@www-template/ui/lib/utils";

	let {
		ref = $bindable(null),
		variant = "outline",
		size = "sm",
		class: className = "",
		...restProps
	}: DropdownMenuPrimitive.TriggerProps & {
		variant?: ButtonVariant;
		size?: ButtonSize;
	} = $props();
</script>

<DropdownMenuPrimitive.Trigger
	bind:ref
	data-slot="dropdown-menu-trigger"
	class={cn(buttonVariants({ variant, size }), className)}
	{...restProps}
/>
