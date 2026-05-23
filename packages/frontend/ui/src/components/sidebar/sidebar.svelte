<script lang="ts">
	import * as Sheet from "@www-template/ui/components/sheet";
	import { cn, type WithElementRef } from "@www-template/ui/lib/utils";
	import type { HTMLAttributes } from "svelte/elements";
	import { SIDEBAR_WIDTH_MOBILE } from "./constants.js";
	import { useSidebar } from "./context.svelte.js";

	let {
		ref = $bindable(null),
		side = "left",
		variant = "sidebar",
		collapsible = "offcanvas",
		ariaLabel,
		closeLabel,
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		side?: "left" | "right";
		variant?: "sidebar" | "floating" | "inset";
		collapsible?: "offcanvas" | "icon" | "none";
		ariaLabel: string;
		closeLabel: string;
	} = $props();

	const sidebar = useSidebar();
</script>

{#if collapsible === "none"}
	<div
		class={cn(
			"bg-sidebar text-sidebar-foreground rounded-xl border border-sidebar-border p-3 flex h-full w-(--sidebar-width) flex-col gap-2",
			className
		)}
		bind:this={ref}
		{...restProps}
	>
		{@render children?.()}
	</div>
{:else if sidebar.isMobile}
	<Sheet.Root
		bind:open={() => sidebar.openMobile, (v) => sidebar.setOpenMobile(v)}
		{...restProps}
	>
			<Sheet.Content
				bind:ref
				data-sidebar="sidebar"
				data-slot="sidebar"
				data-mobile="true"
				aria-label={ariaLabel}
				closeLabel={closeLabel}
class={cn(
			"bg-sidebar text-sidebar-foreground w-(--sidebar-width) rounded-xl p-3 [&>button]:hidden",
			className
		)}
			style="--sidebar-width: {SIDEBAR_WIDTH_MOBILE};"
			{side}
		>
			<div class="flex h-full w-full flex-col">
				{@render children?.()}
			</div>
		</Sheet.Content>
	</Sheet.Root>
{:else}
	<div
		bind:this={ref}
		class="text-sidebar-foreground group peer hidden md:block"
		data-state={sidebar.state}
		data-collapsible={sidebar.state === "collapsed" ? collapsible : ""}
		data-variant={variant}
		data-side={side}
		data-slot="sidebar"
	>
		<!-- This is what handles the sidebar gap on desktop -->
		<div
			data-slot="sidebar-gap"
			class={cn(
				"transition-[width] duration-300 ease-out relative w-(--sidebar-width) bg-transparent",
				"group-data-[collapsible=offcanvas]:w-0",
				"group-data-[side=right]:rotate-180",
				variant === "floating" || variant === "inset"
					? "group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4)))]"
					: "group-data-[collapsible=icon]:w-(--sidebar-width-icon)"
			)}
		></div>
		<div
			data-slot="sidebar-container"
			class={cn(
				"fixed inset-y-0 z-10 hidden h-svh w-(--sidebar-width) transition-[left,right,width] duration-300 ease-out md:flex",
				side === "left"
					? "start-0 group-data-[collapsible=offcanvas]:start-[calc(var(--sidebar-width)*-1)]"
					: "end-0 group-data-[collapsible=offcanvas]:end-[calc(var(--sidebar-width)*-1)]",
				variant === "floating" || variant === "inset"
					? "p-3 group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]"
					: "group-data-[collapsible=icon]:w-(--sidebar-width-icon)",
				className
			)}
			{...restProps}
		>
			<div
				data-sidebar="sidebar"
				data-slot="sidebar-inner"
				class={cn(
					"bg-sidebar flex size-full flex-col",
					variant === "floating" || variant === "inset"
						? "rounded-md border border-sidebar-border p-3 gap-3"
						: "border-r border-sidebar-border"
				)}
			>
				{@render children?.()}
			</div>
		</div>
	</div>
{/if}
