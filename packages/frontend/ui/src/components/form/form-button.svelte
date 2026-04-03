<script lang="ts">
	import { Button, type ButtonProps } from "@ui/components/button/index.js";

	let { ref = $bindable(null), ...restProps }: ButtonProps = $props();
</script>

<Button bind:ref type="submit" {...restProps} />
