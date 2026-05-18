<script lang="ts">
	import {
		AlertDialog,
		AlertDialogAction,
		AlertDialogCancel,
		AlertDialogContent,
		AlertDialogDescription,
		AlertDialogFooter,
		AlertDialogHeader,
		AlertDialogTitle,
		AlertDialogTrigger,
	} from "@www-template/ui/components/alert-dialog";
	import type { ButtonVariant } from "@www-template/ui/components/button";
	import type { Snippet } from "svelte";

	interface Props {
		open?: boolean;
		trigger?: Snippet;
		title?: string;
		description?: string;
		confirmText?: string;
		cancelText?: string;
		confirmVariant?: ButtonVariant;
		onConfirm?: () => void;
		onCancel?: () => void;
		children?: Snippet;
	}

	let {
		open = $bindable(false),
		trigger,
		title,
		description,
		confirmText = '確認',
		cancelText = 'キャンセル',
		confirmVariant = 'default',
		onConfirm,
		onCancel,
		children,
	}: Props = $props();

	function handleConfirm() {
		open = false;
		onConfirm?.();
	}

	function handleCancel() {
		open = false;
		onCancel?.();
	}
</script>

<AlertDialog bind:open>
	{#if trigger}
		<AlertDialogTrigger>
			{@render trigger()}
		</AlertDialogTrigger>
	{/if}
	<AlertDialogContent>
		<AlertDialogHeader>
			{#if title}
				<AlertDialogTitle>{title}</AlertDialogTitle>
			{/if}
			{#if description}
				<AlertDialogDescription>{description}</AlertDialogDescription>
			{/if}
			{#if children}
				{@render children()}
			{/if}
		</AlertDialogHeader>
		<AlertDialogFooter>
			<AlertDialogCancel onclick={handleCancel}>{cancelText}</AlertDialogCancel>
			<AlertDialogAction variant={confirmVariant} onclick={handleConfirm}>
				{confirmText}
			</AlertDialogAction>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>
