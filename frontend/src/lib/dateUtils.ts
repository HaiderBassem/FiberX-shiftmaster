import { format } from 'date-fns';

/**
 * Parse a timestamp from the Go backend.
 *
 * pgx returns timestamptz as Go time.Time, which JSON-serializes as UTC with 'Z'.
 * Example: "2026-05-05T19:21:00Z" → should display as 22:21 (UTC+3/Baghdad)
 *
 * However, some fields may come without 'Z' (plain timestamps).
 * We always ensure UTC interpretation by appending 'Z' when missing.
 */
export function parseTimestamp(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  // Already has explicit timezone info → parse as-is
  if (dateStr.includes('Z') || dateStr.includes('+') || dateStr.includes('-', 10)) {
    return new Date(dateStr);
  }
  // No timezone suffix → append Z to treat as UTC (Go pgx behavior)
  return new Date(dateStr + 'Z');
}

/**
 * Parse a date-only string (YYYY-MM-DD).
 * Uses local noon to avoid day-boundary shifts in UTC+3.
 */
export function parseDate(dateStr: string | null | undefined): Date {
  if (!dateStr) return new Date();
  const datePart = dateStr.split('T')[0];
  return new Date(datePart + 'T12:00:00');
}

/** Format a DB timestamp (date + time) — displays in browser local timezone */
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
