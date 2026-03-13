<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import CardFooter from '@ui/components/molecules/Card/CardFooter.svelte';
  import CardHeader from '@ui/components/molecules/Card/CardHeader.svelte';

  interface Props {
    width?: string | number;
    height?: string | number;
    padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
    variant?: 'default' | 'unstyled';
  }

  let {
    width = '300px',
    height,
    padding = 'none',
    variant = 'default',
  }: Props = $props();
</script>

<Card {width} {height} {padding} {variant}>
  <CardHeader>Card Header</CardHeader>
  <CardBody>This is the card body content. It has default padding and styling.</CardBody>
  <CardFooter>
    <Button
      variant="primary"
      size="sm"
      type="button"
      onclick={() => {
        window.location.hash = 'card-action';
      }}
    >
      Action
    </Button>
  </CardFooter>
</Card>
