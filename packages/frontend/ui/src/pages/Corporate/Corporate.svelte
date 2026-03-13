<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Col from '@ui/components/atoms/Grid/Col.svelte';
  import Container from '@ui/components/atoms/Grid/Container.svelte';
  import Row from '@ui/components/atoms/Grid/Row.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import CardHeader from '@ui/components/molecules/Card/CardHeader.svelte';
  import Footer from '@ui/components/navigation/Footer/Footer.svelte';
  import SiteHeader from '@ui/components/navigation/Header/SiteHeader.svelte';
  import WebsiteLayout from '@ui/layouts/WebsiteLayout/WebsiteLayout.svelte';

  import styles from './Corporate.module.scss';

  const navigationLinks = [
    { label: 'Home', href: '#home', active: true },
    { label: 'About', href: '#about' },
    { label: 'Services', href: '#services' },
    { label: 'Contact', href: '#contact' },
  ];

  const footerLinks = [
    { label: 'Privacy', href: '/privacy' },
    { label: 'Terms', href: '/terms' },
  ];

  function handleLearnMore(): void {
    document.getElementById('services')?.scrollIntoView({
      behavior: 'smooth',
      block: 'start',
    });
  }
</script>

{#snippet header()}
  <SiteHeader links={navigationLinks} />
{/snippet}

{#snippet footer()}
  <Footer links={footerLinks} />
{/snippet}

<WebsiteLayout {header} {footer}>
  <section id="home" class={styles.hero ?? ''}>
    <Container>
      <h1 class={styles.heroTitle ?? ''}>Innovating for the Open Future</h1>
      <p class={styles.heroSubtitle ?? ''}>
        Example Company connects technology and people through open source solutions and modern design.
      </p>
      <Button size="lg" onclick={handleLearnMore}>Learn More</Button>
    </Container>
  </section>

  <section id="about" class={styles.section ?? ''}>
    <Container>
      <h2 class={styles.sectionTitle ?? ''}>About Example Company</h2>
      <p class={styles.heroSubtitle ?? ''}>
        We help teams deliver reliable products by combining strategic consulting, modern engineering,
        and human-centered design.
      </p>
    </Container>
  </section>

  <section id="services" class={styles.section ?? ''}>
    <Container>
      <h2 class={styles.sectionTitle ?? ''}>Our Services</h2>
      <Row>
        <Col>
          <Card>
            <CardHeader>Consulting</CardHeader>
            <CardBody>Strategic IT consulting to help your business grow and adapt.</CardBody>
          </Card>
        </Col>
        <Col>
          <Card>
            <CardHeader>Development</CardHeader>
            <CardBody>Web and mobile application development with cutting-edge tech.</CardBody>
          </Card>
        </Col>
        <Col>
          <Card>
            <CardHeader>Design</CardHeader>
            <CardBody>UI/UX design that puts the user first and looks amazing.</CardBody>
          </Card>
        </Col>
      </Row>
    </Container>
  </section>

  <section id="contact" class={styles.section ?? ''}>
    <Container>
      <h2 class={styles.sectionTitle ?? ''}>Contact</h2>
      <p class={styles.heroSubtitle ?? ''}>
        Reach out to our team at contact@example-company.test and we will respond within one business day.
      </p>
    </Container>
  </section>
</WebsiteLayout>
