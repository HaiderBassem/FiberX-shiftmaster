import { format } from 'date-fns';

/**
 * Parse a timestamp string from the backend.
 * The Go/PostgreSQL backend returns timestamps already in local time
 * (no timezone suffix needed). Using new Date() directly is correct.
 */
export function parseTimestamp(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  return new Date(dateStr);
}

/**
 * Parse a date-only string (YYYY-MM-DD).
 * Appends T12:00:00 (local noon) to avoid day-boundary shifts
 * that occur when JS parses bare ISO dates as UTC midnight.
 */
export function parseDate(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  const datePart = dateStr.split('T')[0];
  return new Date(datePart + 'T12:00:00');
}

/** Format a DB timestamp (date + time) in the browser's local timezone */
export function fmtDateTime(
  dateStr: string | null | undefined,
  fmt = 'MMM d, yyyy · h:mm a'
): string {
  if (!dateStr) return '—';
  try {
    return format(parseTimestamp(dateStr), fmt);
  } catch {
    return '—';
  }
}

/** Format a DB date-only field */
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
