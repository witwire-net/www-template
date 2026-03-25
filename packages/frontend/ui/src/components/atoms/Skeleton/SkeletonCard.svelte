<script lang="ts">
  import Skeleton from './Skeleton.svelte';
</script>

<div style="width: 300px; padding: 16px; border: 1px solid var(--color-border); border-radius: 8px;">
  <div style="display: flex; gap: 16px; margin-bottom: 16px;">
    <Skeleton variant="circular" width={40} height={40} />
    <div style="flex: 1;">
      <Skeleton variant="text" width="80%" />
      <Skeleton variant="text" width="40%" />
    </div>
  </div>
  <Skeleton variant="rectangular" width="100%" height={150} />
</div>
