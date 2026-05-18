<script lang="ts">
	import * as Tooltip from "@www-template/ui/components/tooltip";
	import {
		cn,
		type WithElementRef,
		type WithoutChildrenOrChild,
	} from "@www-template/ui/lib/utils";
	import { mergeProps } from "bits-ui";
	import type { ComponentProps, Snippet } from "svelte";
	import type { HTMLAnchorAttributes } from "svelte/elements";
	import {
		sidebarMenuButtonVariants,
		type SidebarMenuButtonVariant,
		type SidebarMenuButtonSize,
	} from "./sidebar-menu-button.svelte";
	import { useSidebar } from "./context.svelte.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		child,
		variant = "default",
		size = "default",
		isActive = false,
		href,
		tooltipContent,
		tooltipContentProps,
		...restProps
	}: WithElementRef<HTMLAnchorAttributes, HTMLAnchorElement> & {
		isActive?: boolean;
		variant?: SidebarMenuButtonVariant;
		size?: SidebarMenuButtonSize;
		tooltipContent?: Snippet | string;
		tooltipContentProps?: WithoutChildrenOrChild<ComponentProps<typeof Tooltip.Content>>;
		child?: Snippet<[{ props: Record<string, unknown> }]>
	} = $props();

	const sidebar = useSidebar();

	const linkProps = $derived({
		class: cn(sidebarMenuButtonVariants({ variant, size }), className),
		"data-slot": "sidebar-menu-link",
		"data-sidebar": "menu-button",
		"data-size": size,
		"data-active": isActive,
		...restProps,
	});
</script>

{#snippet Link({ props }: { props?: Record<string, unknown> })}
	{@const mergedProps = mergeProps(linkProps, props)}
	{#if child}
		{@render child({ props: mergedProps })}
	{:else}
		<a bind:this={ref} {href} {...mergedProps}>
			{@render children?.()}
		</a>
	{/if}
{/snippet}

{#if !tooltipContent}
	{@render Link({})}
{:else}
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				{@render Link({ props })}
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content
			side="right"
			align="center"
			hidden={sidebar.state !== "collapsed" || sidebar.isMobile}
			{...tooltipContentProps}
		>
			{#if typeof tooltipContent === "string"}
				{tooltipContent}
			{:else if tooltipContent}
				{@render tooltipContent()}
			{/if}
		</Tooltip.Content>
	</Tooltip.Root>
{/if}
