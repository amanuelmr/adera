// Field rules the API also enforces (internal/reviews Input.Validate),
// checked here so the user hears about them on the step they're on, not at
// submit time three steps later.

/** experience_date: optional, a real YYYY-MM-DD calendar date, not in the future. */
export function experienceDateError(value: string, today: Date = new Date()): 'format' | 'future' | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(trimmed);
  if (!match) return 'format';
  const [, y, m, d] = match.map(Number);
  const date = new Date(Date.UTC(y, m - 1, d));
  // Rejects rollovers like 2026-02-31, which Date would silently turn into March.
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) return 'format';
  const todayUTC = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate());
  return date.getTime() > todayUTC ? 'future' : undefined;
}

// Same bounds as the API: greater than zero, at most 100 million birr.
const MAX_PRICE = 100_000_000;

/**
 * price_paid: optional, a positive amount. Accepts the ways people type
 * prices — "1,500", "1 500", "250.50" — rather than silently dropping them.
 */
export function parsePrice(value: string): { amount?: number; error?: 'invalid' } {
  const normalized = value.replace(/[\s,]/g, '');
  if (!normalized) return {};
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return { error: 'invalid' };
  const amount = Number(normalized);
  if (amount <= 0 || amount > MAX_PRICE) return { error: 'invalid' };
  return { amount };
}
