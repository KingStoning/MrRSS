import { describe, expect, it } from 'vitest';
import { getProxiedMediaUrl } from './mediaProxy';

describe('media proxy URL encoding', () => {
  it('round trips UTF-8, signed query strings and base64 plus characters', () => {
    const original = 'https://cdn.example.com/图片.png?a=>>>+&signature=a/b+c==';
    const result = new URL(
      getProxiedMediaUrl(original, 'https://example.com/文章'),
      location.origin
    );
    const decode = (value: string) =>
      new TextDecoder().decode(Uint8Array.from(atob(value), (c) => c.charCodeAt(0)));
    expect(decode(result.searchParams.get('url_b64')!)).toBe(original);
    expect(decode(result.searchParams.get('referer_b64')!)).toBe('https://example.com/文章');
  });
  it('does not wrap an existing local proxy URL again', () => {
    const url = '/api/media/proxy?url_b64=YWJj';
    expect(getProxiedMediaUrl(url, 'https://example.com')).toBe(url);
  });
});
