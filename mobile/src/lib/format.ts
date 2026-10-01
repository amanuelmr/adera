import i18n from './i18n';

// Dates and numbers follow the app's chosen language, not the device's.
// Always Gregorian with Western digits (docs/frontend-handoff.md §7): the
// Ethiopian calendar is an open design question, and reviews are compared
// across users, so everyone should see the same date.
function locale(language: string = i18n.language): string {
  return language === 'am' ? 'am-ET' : 'en-US';
}

const BASE: Intl.DateTimeFormatOptions = { calendar: 'gregory', numberingSystem: 'latn' };

export function formatDate(value: string | Date, language?: string): string {
  return new Intl.DateTimeFormat(locale(language), { ...BASE, dateStyle: 'medium' }).format(new Date(value));
}

export function formatDateTime(value: string | Date, language?: string): string {
  return new Intl.DateTimeFormat(locale(language), { ...BASE, dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value)
  );
}

export function formatNumber(value: number, language?: string, fractionDigits?: number): string {
  return new Intl.NumberFormat(locale(language), {
    numberingSystem: 'latn',
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  } as Intl.NumberFormatOptions).format(value);
}

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 365 * 24 * 3600],
  ['month', 30 * 24 * 3600],
  ['week', 7 * 24 * 3600],
  ['day', 24 * 3600],
  ['hour', 3600],
  ['minute', 60],
];

/**
 * "3 days ago", falling back to the absolute date where the JS engine lacks
 * Intl.RelativeTimeFormat. Pair with formatDate/formatDateTime for an
 * accessible absolute equivalent (docs/frontend-handoff.md §6).
 */
export function formatRelative(value: string | Date, language?: string, now: Date = new Date()): string {
  const date = new Date(value);
  if (typeof Intl.RelativeTimeFormat !== 'function') return formatDate(date, language);
  const seconds = Math.round((date.getTime() - now.getTime()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(locale(language), { numeric: 'auto' });
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit);
  }
  return rtf.format(0, 'minute');
}
