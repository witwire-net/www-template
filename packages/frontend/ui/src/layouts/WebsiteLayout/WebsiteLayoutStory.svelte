<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Col from '@ui/components/atoms/Grid/Col.svelte';
  import Container from '@ui/components/atoms/Grid/Container.svelte';
  import Row from '@ui/components/atoms/Grid/Row.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import Footer from '@ui/components/navigation/Footer/Footer.svelte';
  import SiteHeader from '@ui/components/navigation/Header/SiteHeader.svelte';

  import WebsiteLayout from './WebsiteLayout.svelte';

  let { rich = false }: { rich?: boolean } = $props();

  const defaultLinks = [
    { label: 'Home', href: '/home', active: true },
    { label: 'About', href: '/about' },
    { label: 'Services', href: '/services' },
    { label: 'Contact', href: '/contact' },
  ];

  const richLinks = [
    { label: 'Products', href: '/products' },
    { label: 'Solutions', href: '/solutions' },
    { label: 'Resources', href: '/resources' },
    { label: 'Pricing', href: '/pricing' },
  ];

  function scrollToSection(id: string): void {
    document.getElementById(id)?.scrollIntoView({
      behavior: 'smooth',
      block: 'start',
    });
  }
</script>

{#snippet richBrand()}
  <div style="display: flex; align-items: center; gap: 8px; color: var(--color-primary);">
    <div style="width: 32px; height: 32px; background: currentColor; border-radius: 8px;"></div>
    <span style="font-size: 1.25rem; font-weight: 900;">www-template UI</span>
  </div>
{/snippet}

{#snippet headerActions()}
  <div style="display: flex; gap: 8px;">
    <Button
      variant="ghost"
      size="sm"
      onclick={() => {
        window.location.hash = 'login';
      }}
    >
      Sign In
    </Button>
    <Button
      size="sm"
      onclick={() => {
        window.location.hash = 'signup';
      }}
    >
      Get Started
    </Button>
  </div>
{/snippet}

<WebsiteLayout overlayHeader={true}>
  {#snippet header()}
    <SiteHeader
      actions={rich ? headerActions : undefined}
      links={rich ? richLinks : defaultLinks}
      logo={rich ? richBrand : undefined}
    />
  {/snippet}

  {#snippet footer()}
    <Footer
      links={[
        { label: 'Privacy', href: '/privacy' },
        { label: 'Terms', href: '/terms' },
      ]}
    />
  {/snippet}

  <section
    id="hero"
    style="padding: 5rem 0; background: linear-gradient(135deg, var(--color-background) 0%, #fff 100%);"
  >
    <Container>
      <div style="max-width: 800px; margin: 0 auto; text-align: center;">
        <div style="margin-bottom: 1rem; color: var(--color-primary); font-weight: 700;">Launch Fast</div>
        <h1 style="margin-bottom: 1.5rem; font-size: 3rem; font-weight: 900; line-height: 1.2;">
          Build Beautiful Interfaces with www-template UI
        </h1>
        <p style="margin-bottom: 2.5rem; font-size: 1.25rem; color: var(--color-text-muted);">
          A complete design system for modern web applications. Fast, accessible, and stunningly beautiful.
        </p>
        <div style="display: flex; justify-content: center; gap: 1rem; flex-wrap: wrap;">
          <Button
            size="lg"
            onclick={() => {
              scrollToSection('features');
            }}
          >
            Get Started Now
          </Button>
          <Button
            size="lg"
            variant="outline"
            onclick={() => {
              scrollToSection('hero');
            }}
          >
            View Documentation
          </Button>
        </div>
      </div>
    </Container>
  </section>

  <section id="features" style="padding: 5rem 0;">
    <Container>
      <Row>
        <Col>
          <Card variant="unstyled" style="height: 100%;">
            <CardBody>
              <div style="margin-bottom: 1rem; font-size: 2rem;">⚡️</div>
              <h2 style="margin-bottom: 0.75rem; font-size: 1.25rem;">Fast Performance</h2>
              <p style="color: var(--color-text-muted);">Optimized for speed and efficiency.</p>
            </CardBody>
          </Card>
        </Col>
        <Col>
          <Card variant="unstyled" style="height: 100%;">
            <CardBody>
              <div style="margin-bottom: 1rem; font-size: 2rem;">🎨</div>
              <h2 style="margin-bottom: 0.75rem; font-size: 1.25rem;">Modern Design</h2>
              <p style="color: var(--color-text-muted);">Clean aesthetics that fit any brand.</p>
            </CardBody>
          </Card>
        </Col>
        <Col>
          <Card variant="unstyled" style="height: 100%;">
            <CardBody>
              <div style="margin-bottom: 1rem; font-size: 2rem;">📱</div>
              <h2 style="margin-bottom: 0.75rem; font-size: 1.25rem;">Responsive</h2>
              <p style="color: var(--color-text-muted);">Looks great on any device, anywhere.</p>
            </CardBody>
          </Card>
        </Col>
      </Row>
    </Container>
  </section>
</WebsiteLayout>
