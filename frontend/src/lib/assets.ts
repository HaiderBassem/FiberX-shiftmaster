/**
 * Builds URLs for files served by the API (profile photos, announcement images,
 * rich-text attachments).
 *
 * This expression used to be copy-pasted into six components, each reconstructing
 * an origin from VITE_API_URL and guessing whether the stored path already began
 * with `/api`. Getting it wrong produced a broken image rather than an error, so
 * the copies had quietly drifted.
 *
 * Uploads are authorised by a cookie scoped to `/api/uploads`, which the browser
 * only attaches to same-origin requests. Same-origin is therefore a requirement,
 * not a preference: in production Caddy serves the SPA and proxies `/api` from
 * one origin, and the Vite dev server proxies `/api` for the same reason.
 */

/** Absolute URLs and data URIs are returned untouched. */
function isAbsolute(path: string): boolean {
  return /^(https?:)?\/\//i.test(path) || path.startsWith('data:') || path.startsWith('blob:');
}

/**
 * Resolves a stored file path to a URL the browser can request.
 *
 * Accepts the shapes actually present in the database: `/api/uploads/...`,
 * `/uploads/...`, and bare filenames already prefixed elsewhere.
 * Returns an empty string for empty input so callers can test it directly.
 */
export function assetUrl(storedPath?: string | null): string {
  if (!storedPath) return '';
  if (isAbsolute(storedPath)) return storedPath;

  const path = storedPath.startsWith('/') ? storedPath : `/${storedPath}`;

  // Stored paths may or may not already carry the /api prefix.
  const withApiPrefix = path.startsWith('/api/') ? path : `/api${path}`;

  // VITE_API_URL is normally unset, so requests go to the serving origin. When
  // it is set to an absolute URL, derive the origin from it.
  const configured = import.meta.env.VITE_API_URL;
  if (configured && isAbsolute(configured)) {
    const origin = configured.replace(/\/api\/?$/, '');
    return `${origin}${withApiPrefix}`;
  }

  return withApiPrefix;
}

/**
 * Convenience wrapper for profile photos, returning null when there is none so
 * callers can fall back to initials.
 */
export function profileImageUrl(storedPath?: string | null): string | null {
  const url = assetUrl(storedPath);
  return url === '' ? null : url;
}
