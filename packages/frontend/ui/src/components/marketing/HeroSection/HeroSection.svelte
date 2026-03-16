<script lang="ts">
  import styles from './HeroSection.module.scss';

  interface HeroSectionProps {
    className?: string;
  }

  const { className }: HeroSectionProps = $props();

  /**
   * Word segments to control line-break positions.
   * Each segment stays on a single line (no mid-word breaks).
   */
  const segments = ['機知に', '富んだ', 'おもしろ', 'イノベーション'] as const;
  const accentChars = new Set(['機', '富', 'お', 'イ']);

  let globalIndex = 0;
  const wordGroups = segments.map((segment) => ({
    segment,
    chars: segment.split('').map((char) => ({
      char,
      globalIndex: globalIndex++,
      isAccent: accentChars.has(char),
    })),
  }));

  const joinClassNames = (...values: (string | undefined)[]): string => {
    return values.filter((v) => v !== undefined && v !== '').join(' ');
  };

  const rootClassName = $derived(joinClassNames(styles.root, className));

  /**
   * Data stream characters — monospace ticker for edge decoration.
   * Repeated twice for seamless scrolling loop.
   */
  const dataChars = '01001110 WitWire 11010011 SNS 00101101 BaaS 10110010 API 01011010 ';
  // 1コピー分を十分な長さにする（2コピー縦並びでシームレスループ）
  const dataStreamText = dataChars.repeat(6);
</script>

<section class={rootClassName} id="hero">
  <!-- Layer 4: Signal horizon -->
  <div class={styles.signalField ?? ''}>
    <div class={styles.signalGlow ?? ''}></div>
    <div class={styles.signalHorizon ?? ''}></div>
    <div class={styles.signalPlane ?? ''}></div>
  </div>

  <!-- Layer 5: Circuit traces (left) -->
  <div class={styles.circuitLeft ?? ''}>
    <div class={joinClassNames(styles.tracePath, styles.traceLH1)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLH2)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLH3)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLH4)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLH5)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLV1)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLV2)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLV3)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceLV4)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeL1)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeL2)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeL3)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeL4)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeL5)}></div>
  </div>

  <!-- Layer 5: Circuit traces (right) -->
  <div class={styles.circuitRight ?? ''}>
    <div class={joinClassNames(styles.tracePath, styles.traceRH1)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRH2)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRH3)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRH4)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRH5)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRV1)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRV2)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRV3)}></div>
    <div class={joinClassNames(styles.tracePath, styles.traceRV4)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeR1)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeR2)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeR3)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeR4)}></div>
    <div class={joinClassNames(styles.traceNode, styles.nodeR5)}></div>
  </div>

  <!-- Layer 6: Floating geometry -->
  <div class={styles.floatingGeo ?? ''}>
    <div class={joinClassNames(styles.geoShape, styles.geoTriangleInner)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoCircle)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoSquare)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoDiamond)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoCircleSm)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoSquareSm)}></div>
    <div class={joinClassNames(styles.geoShape, styles.geoTriangle2)}></div>
    <div class={styles.geoCross ?? ''}></div>
  </div>

  <!-- Layer 7: Data streams (edge typography) -->
  <div class={joinClassNames(styles.dataStream, styles.dataStreamLeft)}>
    <div class={styles.dataStreamTrack ?? ''}>
      <span class={styles.dataStreamCopy ?? ''}>{dataStreamText}</span>
      <span class={styles.dataStreamCopy ?? ''}>{dataStreamText}</span>
    </div>
  </div>
  <div class={joinClassNames(styles.dataStream, styles.dataStreamRight)}>
    <div class={styles.dataStreamTrack ?? ''}>
      <span class={styles.dataStreamCopy ?? ''}>{dataStreamText}</span>
      <span class={styles.dataStreamCopy ?? ''}>{dataStreamText}</span>
    </div>
  </div>

  <!-- Layer 8: Top decoration lines -->
  <div class={styles.topLines ?? ''}>
    <div class={joinClassNames(styles.topLine, styles.topLine1)}></div>
    <div class={joinClassNames(styles.topLine, styles.topLine2)}></div>
    <div class={joinClassNames(styles.topLine, styles.topLine3)}></div>
  </div>

  <!-- Layer 9: Concentric rings (behind text) -->
  <div class={styles.concentricRings ?? ''}>
    <div class={joinClassNames(styles.ring, styles.ring1)}></div>
    <div class={joinClassNames(styles.ring, styles.ring2)}></div>
    <div class={joinClassNames(styles.ring, styles.ring3)}></div>
  </div>

  <!-- Corner brackets -->
  <div class={joinClassNames(styles.cornerDecor, styles.cornerTL)}></div>
  <div class={joinClassNames(styles.cornerDecor, styles.cornerTR)}></div>
  <div class={joinClassNames(styles.cornerDecor, styles.cornerBL)}></div>
  <div class={joinClassNames(styles.cornerDecor, styles.cornerBR)}></div>

  <!-- Content -->
  <div class={styles.content ?? ''}>
    <h1 class={styles.title ?? ''}>
      {#each wordGroups as { segment, chars } (segment)}
        <span class={styles.wordGroup ?? ''}>
          {#each chars as { char, globalIndex: gi, isAccent } (gi)}
            <span
              class={isAccent
                ? joinClassNames(styles.char, styles.accent)
                : (styles.char ?? '')}
              style="--char-index: {gi};"
            >
              {char}
            </span>
          {/each}
        </span>
      {/each}
    </h1>
    <span class={styles.underline ?? ''}></span>
  </div>
</section>
