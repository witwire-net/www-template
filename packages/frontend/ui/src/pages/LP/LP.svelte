<svelte:options runes={true} />

<script lang="ts">
  import { IconBolt, IconDeviceMobile, IconPalette } from '@tabler/icons-svelte';

  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Col from '@ui/components/atoms/Grid/Col.svelte';
  import Container from '@ui/components/atoms/Grid/Container.svelte';
  import Row from '@ui/components/atoms/Grid/Row.svelte';
  import Icon from '@ui/components/atoms/Icon/Icon.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import Footer from '@ui/components/navigation/Footer/Footer.svelte';
  import SiteHeader from '@ui/components/navigation/Header/SiteHeader.svelte';
  import WebsiteLayout from '@ui/layouts/WebsiteLayout/WebsiteLayout.svelte';

  import styles from './LP.module.scss';

  type FeatureCard = {
    description: string;
    icon: typeof IconBolt;
    title: string;
    titleText: string;
  };

  const featureCards: readonly FeatureCard[] = [
    {
      icon: IconBolt,
      title: 'Fast performance',
      titleText: 'Fast Performance',
      description: 'Optimized for speed and efficiency.',
    },
    {
      icon: IconPalette,
      title: 'Modern design',
      titleText: 'Modern Design',
      description: 'Clean aesthetics that fit any brand.',
    },
    {
      icon: IconDeviceMobile,
      title: 'Responsive',
      titleText: 'Responsive',
      description: 'Looks great on any device, anywhere.',
    },
  ] as const;

  function scrollToSection(id: string): void {
    document.getElementById(id)?.scrollIntoView({
      behavior: 'smooth',
      block: 'start',
    });
  }
</script>

{#snippet header()}
  <SiteHeader />
{/snippet}

{#snippet footer()}
  <Footer />
{/snippet}

<WebsiteLayout {header} {footer}>
  <section id="hero" class={styles.hero ?? ''}>
    <Container>
      <div class={styles.heroContent ?? ''}>
        <div class={styles.tagline ?? ''}>Launch Fast</div>
        <h1 class={styles.title ?? ''}>Build Beautiful Interfaces with www-template UI</h1>
        <p class={styles.description ?? ''}>
          A complete design system for modern web applications. Fast, accessible, and stunningly
          beautiful.
        </p>
        <div class={styles.ctas ?? ''}>
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
              scrollToSection('documentation');
            }}
          >
            View Documentation
          </Button>
        </div>
      </div>
    </Container>
  </section>

  <section id="features" class={styles.features ?? ''}>
    <Container>
      <Row>
        {#each featureCards as feature (feature.title)}
          <Col>
            <Card className={styles.featureCard ?? ''} variant="unstyled">
              <CardBody>
                <Icon
                  icon={feature.icon}
                  className={styles.featureIcon ?? ''}
                  size={34}
                  title={feature.title}
                />
                <h3>{feature.titleText}</h3>
                <p>{feature.description}</p>
              </CardBody>
            </Card>
          </Col>
        {/each}
      </Row>
    </Container>
  </section>

  <section id="documentation" class={styles.documentation ?? ''}>
    <Container>
      <div class={styles.documentationContent ?? ''}>
        <h2 class={styles.documentationTitle ?? ''}>Documentation</h2>
        <p class={styles.documentationDescription ?? ''}>
          Explore component APIs, accessibility guidelines, and layout recipes to speed up
          implementation.
        </p>
        <a class={styles.documentationLink ?? ''} href="/storybook">Open Docs Portal</a>
      </div>
    </Container>
  </section>
</WebsiteLayout>
