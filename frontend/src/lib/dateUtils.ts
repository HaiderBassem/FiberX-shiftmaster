import { format } from 'date-fns';

// ─── Baghdad timezone (UTC+3) ──────────────────────────────────────────────
// The Go/pgx backend stores and returns ALL timestamps in UTC.
// The frontend is responsible for converting to Asia/Baghdad for display.
// Rule: ALWAYS use fmtDateTime / fmtDate from this file — never format
// backend timestamps with raw JS Date methods or date-fns directly.
const BAGHDAD_TZ = 'Asia/Baghdad';

/**
 * Normalize any backend timestamp string to a proper JS Date (UTC).
 *
 * The backend (pgx v5 + PostgreSQL session TZ=UTC) sends timestamps as:
 *   "2026-05-31T20:26:00Z"        — ISO 8601 with Z suffix (UTC)        ← new records
 *   "2026-05-31T20:26:00+00:00"   — ISO 8601 with +00:00 offset         ← also valid
 *   "2026-05-31T23:26:00+03:00"   — ISO 8601 with Baghdad offset         ← edge case
 *   "2026-05-31T23:26:00"         — no suffix (legacy data stored as     ← old records
 *                                    Baghdad wall-clock but labeled UTC)
 *
 * For the no-suffix case: JS spec says it's *local* time, which means it
 * shifts with the browser/OS timezone — unreliable. We force it to be
 * treated as UTC (by appending Z) so the Intl conversion is always correct.
 */
function toUTC(dateStr: string): Date {
  // Already has a timezone marker — parse as-is
  if (/[Z+\-]\d{0,2}:?\d{0,2}$/.test(dateStr.trim()) && dateStr.trim() !== '') {
    return new Date(dateStr);
  }
  // No timezone marker — treat as UTC (append Z)
  return new Date(dateStr + 'Z');
}

/**
 * Format a backend timestamp as Baghdad date + time.
 * Works regardless of the user's browser/OS timezone.
 *
 * Example output: "May 31, 2026, 11:26 PM"
 */
export function fmtDateTime(
  dateStr: string | null | undefined,
  _fmt?: string   // kept for API compatibility — Intl is used internally
): string {
  if (!dateStr) return '—';
  try {
    const date = toUTC(dateStr);
    if (isNaN(date.getTime())) return '—';
    return new Intl.DateTimeFormat('en-US', {
      timeZone: BAGHDAD_TZ,
      year:     'numeric',
      month:    'short',
      day:      'numeric',
      hour:     'numeric',
      minute:   '2-digit',
      hour12:   true,
    }).format(date);
  } catch {
    return '—';
  }
}

/**
 * Format a backend timestamp as Baghdad time only (no date).
 * Example output: "11:26 PM"
 */
export function fmtTime(dateStr: string | null | undefined): string {
  if (!dateStr) return '—';
  try {
    const date = toUTC(dateStr);
    if (isNaN(date.getTime())) return '—';
    return new Intl.DateTimeFormat('en-US', {
      timeZone: BAGHDAD_TZ,
      hour:     'numeric',
      minute:   '2-digit',
      hour12:   true,
    }).format(date);
  } catch {
    return '—';
  }
}

/**
 * Format a date-only field (no time component shown).
 * Input can be "YYYY-MM-DD" or a full timestamp string.
 * Example output: "May 31, 2026"
 */
export function fmtDate(
  dateStr: string | null | undefined,
  fmt = 'MMM d, yyyy'
): string {
  if (!dateStr) return '—';
  try {
    // For date-only strings (YYYY-MM-DD), append noon UTC to avoid
    // midnight boundary shifts across timezones.
    const datePart = dateStr.split('T')[0];
    return format(new Date(datePart + 'T12:00:00Z'), fmt);
  } catch {
    return '—';
  }
}

/**
 * Returns the current time as a Date object in Baghdad (UTC+3).
 * Use this when you need "now" for Baghdad-local comparisons.
 */
export function nowBaghdad(): Date {
  return new Date(
    new Date().toLocaleString('en-US', { timeZone: BAGHDAD_TZ })
  );
}

/**
 * Parse a timestamp string from the backend.
 * Returns a JS Date correctly anchored in UTC.
 */
export function parseTimestamp(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  return toUTC(dateStr);
}

/**
 * Parse a date-only string (YYYY-MM-DD).
 * Returns a Date at noon UTC to avoid midnight day-boundary shifts.
 */
export function parseDate(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  const datePart = dateStr.split('T')[0];
  return new Date(datePart + 'T12:00:00Z');
}
