// Initials and a stable colour for a person's avatar, from their display
// name. Pure functions so they can be tested without rendering.

/** Up to two initials; works for Ethiopic names, which have no case. */
export function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return '?';
  const first = Array.from(words[0])[0] ?? '';
  const last = words.length > 1 ? (Array.from(words[words.length - 1])[0] ?? '') : '';
  return (first + last).toUpperCase();
}

// Mid-tone backgrounds that all carry white text at 4.5:1 or better.
export const AVATAR_COLORS = ['#0B6E4F', '#7A4A1E', '#1F5A8A', '#8A2F5B', '#5B4A99', '#8A5A00', '#2F6F73'] as const;

/** The same name always gets the same colour. */
export function avatarColorFor(name: string): string {
  let hash = 0;
  for (const ch of name) hash = (hash * 31 + (ch.codePointAt(0) ?? 0)) >>> 0;
  return AVATAR_COLORS[hash % AVATAR_COLORS.length];
}
