import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createArticleImageRecovery } from './articleImageRecovery';
import { isMediaProxyFallbackEnabled } from './mediaProxy';

vi.mock('./mediaProxy', () => ({
  isMediaProxyFallbackEnabled: vi.fn().mockResolvedValue(true),
  getProxiedMediaUrl: (url: string) =>
    url.includes('/api/media/proxy') ? url : '/api/media/proxy?url=' + encodeURIComponent(url),
}));

describe('reader image recovery', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(isMediaProxyFallbackEnabled).mockResolvedValue(true);
  });
  afterEach(() => {
    document.body.innerHTML = '';
    vi.useRealTimers();
    vi.clearAllMocks();
  });
  function image() {
    const img = document.createElement('img');
    img.src = 'https://cdn.example.com/chart.png?signature=unchanged';
    document.body.append(img);
    return img;
  }
  it('recovers a failed inline image through the configured proxy without replacing its node', async () => {
    const recovery = createArticleImageRecovery();
    const img = image();
    recovery.attach(img, 'https://example.com/story');
    recovery.attach(img, 'https://example.com/story');
    img.dispatchEvent(new Event('error'));
    await vi.advanceTimersByTimeAsync(350);
    expect(img.src).toContain('/api/media/proxy');
    expect(document.querySelector('img')).toBe(img);
    expect(isMediaProxyFallbackEnabled).toHaveBeenCalledTimes(1);
    recovery.clear();
  });
  it('honors disabled proxy and preserves signed URLs when retrying', async () => {
    vi.mocked(isMediaProxyFallbackEnabled).mockResolvedValue(false);
    const recovery = createArticleImageRecovery();
    const img = image();
    const original = img.src;
    recovery.attach(img);
    img.dispatchEvent(new Event('error'));
    await vi.advanceTimersByTimeAsync(350);
    expect(img.src).toBe(original);
    recovery.clear();
  });
  it('caps failures at two retries and cancels pending work on article change', async () => {
    const recovery = createArticleImageRecovery();
    const img = image();
    recovery.attach(img);
    for (let i = 0; i < 5; i++) {
      img.dispatchEvent(new Event('error'));
      await vi.advanceTimersByTimeAsync(700);
    }
    expect(isMediaProxyFallbackEnabled).toHaveBeenCalledTimes(2);
    recovery.clear();
    const next = image();
    const original = next.src;
    recovery.attach(next);
    next.dispatchEvent(new Event('error'));
    recovery.clear();
    await vi.advanceTimersByTimeAsync(1000);
    expect(next.src).toBe(original);
  });
});
