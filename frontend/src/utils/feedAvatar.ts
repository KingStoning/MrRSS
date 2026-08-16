const FEED_AVATAR_COLORS = [
  '#7c5cff',
  '#2c8a3e',
  '#0a6bd4',
  '#d23a8b',
  '#a8501f',
  '#ff6600',
  '#c0392b',
  '#d05050',
  '#3a4cb8',
  '#1d8a8a',
] as const;

const CJK_GLYPH = /[\u3040-\u30ff\u3400-\u9fff\uac00-\ud7af\uf900-\ufaff]/;

export function getFeedAvatarLabel(title: string): string {
  const normalizedTitle = title.trim();
  if (!normalizedTitle) return '?';

  const characters = Array.from(normalizedTitle);
  if (CJK_GLYPH.test(characters[0])) return characters[0];

  const words = normalizedTitle.split(/[\s·|—-]+/).filter(Boolean);
  if (words.length > 1) {
    return `${Array.from(words[0])[0] ?? ''}${Array.from(words[1])[0] ?? ''}`.toUpperCase();
  }

  return characters.slice(0, 2).join('').toUpperCase();
}

export function getFeedAvatarColor(seed: string | number): string {
  const hash = Array.from(String(seed)).reduce(
    (value, character) => (value * 31 + character.codePointAt(0)!) >>> 0,
    0
  );
  return FEED_AVATAR_COLORS[hash % FEED_AVATAR_COLORS.length];
}
