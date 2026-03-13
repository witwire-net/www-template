<script lang="ts">
</script>

<section class="overview">
  <div class="eyebrow">AUTH ENTRY</div>
  <h2>認証側の最小導線</h2>
  <p>
    現時点では `/app/profiles` に認証 shell とプレースホルダー案内を集約し、実データ接続は次の差分へ残しています。
  </p>
  <a href="/app/profiles">/app/profiles を開く</a>
</section>

<style>
  .overview {
    display: grid;
    gap: 0.85rem;
  }

  .eyebrow {
    font-size: 0.74rem;
    font-weight: 700;
    letter-spacing: 0.2em;
    color: #1d4ed8;
  }

  h2,
  p {
    margin: 0;
  }

  p {
    color: #475569;
    line-height: 1.6;
  }

  a {
    display: inline-flex;
    align-items: center;
    width: fit-content;
    padding: 0.75rem 1rem;
    border-radius: 999px;
    background: rgba(30, 64, 175, 0.1);
    color: #1d4ed8;
    font-weight: 700;
  }
</style>
