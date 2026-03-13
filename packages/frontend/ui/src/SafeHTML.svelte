<script lang="ts">
  import DOMPurify from 'dompurify';
  import type { Config as DOMPurifyConfig } from 'dompurify';

  import type { SafeHTMLProps } from './SafeHTML.types';

  const defaultConfig: DOMPurifyConfig = {
    ALLOWED_ATTR: [
      'class',
      'id',
      'href',
      'title',
      'target',
      'rel',
      'src',
      'alt',
      'width',
      'height',
      'colspan',
      'rowspan',
      'data-language',
    ],
    ALLOWED_TAGS: [
      'p',
      'br',
      'h1',
      'h2',
      'h3',
      'h4',
      'h5',
      'h6',
      'blockquote',
      'pre',
      'code',
      'strong',
      'b',
      'em',
      'i',
      'u',
      's',
      'del',
      'ins',
      'mark',
      'small',
      'sub',
      'sup',
      'ul',
      'ol',
      'li',
      'a',
      'table',
      'thead',
      'tbody',
      'tfoot',
      'tr',
      'th',
      'td',
      'button',
      'img',
      'hr',
      'span',
      'div',
    ],
    ALLOWED_URI_REGEXP: /^(?:https?|mailto|tel):/i,
    ALLOW_DATA_ATTR: false,
    ALLOW_UNKNOWN_PROTOCOLS: false,
    SANITIZE_DOM: true,
  };

  let {
    html,
    className = undefined,
    sanitizeOptions = undefined,
  }: SafeHTMLProps = $props();

  function removeDataUriImages(markup: string): string {
    if (typeof DOMParser === 'undefined') {
      return markup.replace(/<img\b[^>]*\bsrc\s*=\s*(['"])data:[\s\S]*?\1[^>]*>/giu, '');
    }

    const parser = new DOMParser();
    const documentFragment = parser.parseFromString(markup, 'text/html');

    for (const image of documentFragment.querySelectorAll('img')) {
      const source = image.getAttribute('src');

      if (typeof source === 'string' && source.trim().toLowerCase().startsWith('data:')) {
        image.remove();
      }
    }

    return documentFragment.body.innerHTML;
  }

  function sanitizeMarkup(markup: string, options?: DOMPurifyConfig): string {
    const config = options ?? defaultConfig;
    const sanitized = DOMPurify.sanitize(markup, config);
    const normalizedMarkup = typeof sanitized === 'string' ? sanitized : String(sanitized);

    return removeDataUriImages(normalizedMarkup);
  }

  const finalHTML = $derived.by(() => sanitizeMarkup(html, sanitizeOptions));
</script>

<div class={className}>{@html finalHTML}</div>
