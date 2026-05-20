<script lang="ts">
	import type { ComponentProps } from "svelte";
	import { cn } from "@www-template/ui/lib/utils";
	import { PaginationLink } from "./index.js";
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';

	type PaginationPreviousProps = ComponentProps<typeof PaginationLink>;

	let { class: className, ...restProps }: PaginationPreviousProps = $props();
</script>

<PaginationLink
	size="default"
	class={cn("pl-2!", className)}
	{...restProps}
>
	<ChevronLeftIcon data-icon="inline-start" />
</PaginationLink>
