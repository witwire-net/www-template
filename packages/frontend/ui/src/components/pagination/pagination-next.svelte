<script lang="ts">
	import type { ComponentProps } from "svelte";
	import { cn } from "@www-template/ui/lib/utils";
	import { PaginationLink } from "./index.js";
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	type PaginationNextProps = ComponentProps<typeof PaginationLink>;

	let { class: className, ...restProps }: PaginationNextProps = $props();
</script>

<PaginationLink
	size="default"
	class={cn("pr-2!", className)}
	{...restProps}
>
	<ChevronRightIcon data-icon="inline-end" />
</PaginationLink>
