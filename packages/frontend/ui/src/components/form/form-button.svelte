<script lang="ts">
	import { Button, type ButtonProps } from "@www-template/ui/components/button";

	let { ref = $bindable(null), ...restProps }: ButtonProps = $props();
</script>

<Button bind:ref type="submit" {...restProps} />
