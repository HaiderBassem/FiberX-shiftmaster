import { format } from 'date-fns';

/**
 * Parse a timestamp string as UTC.
 * Go/Postgres sometimes returns timestamps without 'Z', causing JS to treat
 * them as LOCAL time instead of UTC — which produces wrong times in UTC+3.
 * This function always ensures correct UTC interpretation.
 */
export function parseTimestamp(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  // Already has timezone info → parse directly
  if (dateStr.endsWith('Z') || dateStr.includes('+')) {
    return new Date(dateStr);
  }
  // No timezone suffix → assume UTC (add Z)
  return new Date(dateStr + 'Z');
}

/**
 * Parse a date-only string (YYYY-MM-DD or ISO with T).
 * Uses local noon to avoid day-boundary shifts in any timezone.
 */
export function parseDate(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  const datePart = dateStr.split('T')[0];
  return new Date(datePart + 'T12:00:00');
}

/** Format a DB timestamp (with time) using the browser's local timezone */
export function fmtDateTime(dateStr: string | null | undefined, fmt = 'MMM d, yyyy · h:mm a'): string {
  if (!dateStr) return '—';
  try {
    return format(parseTimestamp(dateStr), fmt);
  } catch {
    return '—';
  }
}

/** Format a DB date-only field (no time component) */
export function fmtDate(dateStr: string | null | undefined, fmt = 'MMM d, yyyy'): string {
  if (!dateStr) return '—';
  try {
    return format(parseDate(dateStr), fmt);
  } catch {
    return '—';
  }
}
