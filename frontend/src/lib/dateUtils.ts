import { format } from 'date-fns';

// ─── Baghdad timezone (UTC+3) ──────────────────────────────────────────────
// The Go/pgx backend returns timestamps in Baghdad local time WITHOUT a
// timezone suffix, e.g. "2026-05-05T23:26:00". Adding "Z" incorrectly
// treats them as UTC and shifts them +3 hours. We therefore parse them
// as-is (JavaScript treats no-suffix strings as local time) and then
// format them using Intl with the Baghdad timezone hardcoded so the
// display is always correct regardless of the OS/browser timezone.
const BAGHDAD_TZ = 'Asia/Baghdad';

/**
 * Parse a timestamp string from the backend.
 * Returns a JS Date. Works for both "2026-05-05T23:26:00" (no suffix)
 * and "2026-05-05T20:26:00Z" / "2026-05-05T23:26:00+03:00".
 */
export function parseTimestamp(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  return new Date(dateStr);
}

/**
 * Parse a date-only string (YYYY-MM-DD).
 * Appends T12:00:00 to avoid midnight UTC→local day-boundary shifts.
 */
export function parseDate(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  const datePart = dateStr.split('T')[0];
  return new Date(datePart + 'T12:00:00');
}

/**
 * Format a DB timestamp, always displayed in Baghdad time (UTC+3).
 * This is correct regardless of the user's browser/OS timezone.
 */
export function fmtDateTime(
  dateStr: string | null | undefined,
  _fmt?: string          // kept for API compat; Intl is used internally
): string {
  if (!dateStr) return '—';
  try {
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return '—';
    return new Intl.DateTimeFormat('en-US', {
      timeZone: BAGHDAD_TZ,
      year:    'numeric',
      month:   'short',
      day:     'numeric',
      hour:    'numeric',
      minute:  '2-digit',
      hour12:  true,
    }).format(date);
  } catch {
    return '—';
  }
}

/**
 * Format a DB date-only field (no time component shown).
 */
export function fmtDate(
  dateStr: string | null | undefined,
  fmt = 'MMM d, yyyy'
): string {
  if (!dateStr) return '—';
  try {
    return format(parseDate(dateStr), fmt);
  } catch {
    return '—';
  }
}
