<script lang="ts">
  import Col from './Col.svelte';
  import Container from './Container.svelte';
  import Row from './Row.svelte';

  let { fluid = false }: { fluid?: boolean } = $props();
</script>

<Container {fluid}>
  <Row>
    <Col>
      <div style="padding: 16px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Column A</div>
    </Col>
    <Col>
      <div style="padding: 16px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Column B</div>
    </Col>
    <Col>
      <div style="padding: 16px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Column C</div>
    </Col>
  </Row>
</Container>
