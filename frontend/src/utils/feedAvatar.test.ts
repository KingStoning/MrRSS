import { describe, expect, it } from 'vitest';
import { getFeedAvatarColor, getFeedAvatarLabel } from './feedAvatar';

describe('getFeedAvatarLabel', () => {
  it('uses one glyph for CJK feed names', () => {
    expect(getFeedAvatarLabel('科技日报')).toBe('科');
    expect(getFeedAvatarLabel('ファミ通')).toBe('フ');
  });

  it('uses initials for multi-word Latin feed names', () => {
    expect(getFeedAvatarLabel('Linux Do')).toBe('LD');
  });

  it('handles empty and emoji names without splitting code points', () => {
    expect(getFeedAvatarLabel('')).toBe('?');
    expect(getFeedAvatarLabel('🚀 News')).toBe('🚀N');
  });
});

describe('getFeedAvatarColor', () => {
  it('returns a stable color for a feed id', () => {
    expect(getFeedAvatarColor(42)).toBe(getFeedAvatarColor(42));
    expect(getFeedAvatarColor(42)).toMatch(/^#[0-9a-f]{6}$/i);
  });
});
