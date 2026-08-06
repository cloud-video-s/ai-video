const defaultMediaBaseURL = 'https://test-cdn.zdrawai.com/'

function configuredMediaBaseURL() {
  const configured = String(import.meta.env.VITE_MEDIA_BASE_URL || '').trim()
  const value = configured || defaultMediaBaseURL
  return value.endsWith('/') ? value : `${value}/`
}

/**
 * Keeps absolute URLs intact and expands domain-free upload paths for display.
 * The original half URL must still be used in form payloads and persistence.
 */
export function toMediaURL(rawValue?: string | null): string {
  const value = String(rawValue || '').trim()
  if (!value) return ''
  if (/^[a-z][a-z\d+.-]*:/i.test(value)) return value
  if (value.startsWith('//')) return `https:${value}`
  try {
    return new URL(value.replace(/^\/+/, ''), configuredMediaBaseURL()).toString()
  } catch {
    return value
  }
}
