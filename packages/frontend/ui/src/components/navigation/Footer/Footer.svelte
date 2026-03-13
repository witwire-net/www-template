<script lang="ts">
  import Container from '@ui/components/atoms/Grid/Container.svelte';

  import styles from './Footer.module.scss';

  type FooterLink = {
    label: string;
    href: string;
  };

  type Props = {
    copyright?: string;
    links?: FooterLink[];
  };

  let {
    links = [],
    copyright = `© ${String(new Date().getFullYear())} www-template UI. All rights reserved.`,
  }: Props = $props();
</script>

<footer class={styles.footer ?? ''}>
  <Container>
    <div class={styles.content ?? ''}>
      <div class={styles.copyright ?? ''}>{copyright}</div>
      <div class={styles.links ?? ''}>
        {#each links as link (link.label)}
          <a href={link.href} class={styles.link ?? ''}>{link.label}</a>
        {/each}
      </div>
    </div>
  </Container>
</footer>
