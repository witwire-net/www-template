import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import SafeHTML from './SafeHTML.svelte';

describe('SafeHTML', () => {
  describe('基本的な HTML レンダリング', () => {
    it('安全な HTML を正しくレンダリングする', () => {
      const html = '<p>Hello World</p>';
      render(SafeHTML, { props: { html } });

      expect(screen.queryByText('Hello World')).not.toBeNull();
    });

    it('複数のタグを含む HTML をレンダリングする', () => {
      const html = '<div><h1>Title</h1><p>Content</p></div>';
      render(SafeHTML, { props: { html } });

      expect(screen.queryByText('Title')).not.toBeNull();
      expect(screen.queryByText('Content')).not.toBeNull();
    });

    it('テキスト装飾タグを正しくレンダリングする', () => {
      const html = '<p><strong>Bold</strong> <em>Italic</em></p>';
      const { container } = render(SafeHTML, { props: { html } });

      const strong = container.querySelector('strong');
      const em = container.querySelector('em');

      expect(strong).not.toBeNull();
      expect(strong?.textContent).toBe('Bold');
      expect(em).not.toBeNull();
      expect(em?.textContent).toBe('Italic');
    });
  });

  describe('XSS 攻撃の防御', () => {
    it('script タグを削除する', () => {
      const html = '<p>Safe content</p><script>alert("XSS")</script>';
      const { container } = render(SafeHTML, { props: { html } });

      expect(screen.queryByText('Safe content')).not.toBeNull();
      expect(container.querySelector('script')).toBeNull();
    });

    it('onclick などのイベントハンドラを削除する', () => {
      const html = '<button onclick="alert(\'XSS\')">Click me</button>';
      const { container } = render(SafeHTML, { props: { html } });

      const button = container.querySelector('button');
      expect(button).not.toBeNull();
      expect(button?.getAttribute('onclick')).toBeNull();
    });

    it('危険なプロトコルを含むリンクを削除する', () => {
      const dangerousProtocol = 'javascript';
      const html = `<a href="${dangerousProtocol}:alert('XSS')">Click</a>`;
      const { container } = render(SafeHTML, { props: { html } });

      const link = container.querySelector('a');
      expect(link).not.toBeNull();
      expect(link?.getAttribute('href')).toBeNull();
    });

    it('data: プロトコルを含む画像を削除する', () => {
      const html = '<img src="data:text/html,<script>alert(\'XSS\')</script>" />';
      const { container } = render(SafeHTML, { props: { html } });

      const image = container.querySelector('img');
      expect(image).toBeNull();
    });
  });

  describe('リンクと画像', () => {
    it('安全な URL のリンクをレンダリングする', () => {
      const html = '<a href="https://example.com">Link</a>';
      const { container } = render(SafeHTML, { props: { html } });

      const link = container.querySelector('a');
      expect(link).not.toBeNull();
      expect(link?.getAttribute('href')).toBe('https://example.com');
      expect(link?.textContent).toBe('Link');
    });

    it('安全な画像をレンダリングする', () => {
      const html = '<img src="https://example.com/image.jpg" alt="Test Image" />';
      const { container } = render(SafeHTML, { props: { html } });

      const image = container.querySelector('img');
      expect(image).not.toBeNull();
      expect(image?.getAttribute('src')).toBe('https://example.com/image.jpg');
      expect(image?.getAttribute('alt')).toBe('Test Image');
    });
  });

  describe('カスタムクラス名', () => {
    it('カスタムクラス名を適用する', () => {
      const html = '<p>Content</p>';
      const { container } = render(SafeHTML, { props: { className: 'custom-class', html } });

      const element = container.querySelector('.custom-class');
      expect(element).not.toBeNull();
    });
  });

  describe('カスタムサニタイズ設定', () => {
    it('カスタム設定で許可タグを制限できる', () => {
      const html = '<p>Paragraph</p><strong>Bold</strong>';
      const { container } = render(SafeHTML, {
        props: {
          html,
          sanitizeOptions: { ALLOWED_TAGS: ['p'] },
        },
      });

      expect(screen.queryByText('Paragraph')).not.toBeNull();
      const strong = container.querySelector('strong');
      expect(strong).toBeNull();
    });

    it('カスタム設定で許可属性を制限できる', () => {
      const html = '<a href="https://example.com" title="Example">Link</a>';
      const { container } = render(SafeHTML, {
        props: {
          html,
          sanitizeOptions: {
            ALLOWED_ATTR: ['href'],
            ALLOWED_TAGS: ['a'],
          },
        },
      });

      const link = container.querySelector('a');
      expect(link).not.toBeNull();
      expect(link?.getAttribute('href')).toBe('https://example.com');
      expect(link?.getAttribute('title')).toBeNull();
    });
  });

  describe('マークダウン HTML のサポート', () => {
    it('マークダウンから生成された HTML をレンダリングする', () => {
      const html = `
        <h1>Title</h1>
        <p>This is a paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
        <ul>
          <li>Item 1</li>
          <li>Item 2</li>
        </ul>
      `;
      const { container } = render(SafeHTML, { props: { html } });

      expect(screen.getByRole('heading', { level: 1 }).textContent).toContain('Title');
      expect(screen.queryByText('bold')).not.toBeNull();
      expect(screen.queryByText('italic')).not.toBeNull();

      const listItems = container.querySelectorAll('li');
      expect(listItems).toHaveLength(2);
    });
  });
});
