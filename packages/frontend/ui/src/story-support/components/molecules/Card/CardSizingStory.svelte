<script lang="ts">
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import CardHeader from '@ui/components/molecules/Card/CardHeader.svelte';

  interface Props {
    width?: string | number;
    height?: string | number;
    padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
    variant?: 'default' | 'unstyled';
  }

  let {
    width = '200px',
    height = '200px',
    padding = 'sm',
    variant = 'default',
  }: Props = $props();
</script>

<div style="display: flex; gap: 20px; flex-wrap: wrap;">
  <Card {width} {height} {padding} {variant}>
    <CardBody>Fixed 200x200, Small Padding</CardBody>
  </Card>
  <Card width="300px" padding="lg">
    <CardHeader>Width 300px</CardHeader>
    <CardBody>Auto height, Large Padding</CardBody>
  </Card>
  <Card width="100%" height="150px" padding="md">
    <CardBody>Full Width, Fixed Height 150px</CardBody>
  </Card>
</div>
