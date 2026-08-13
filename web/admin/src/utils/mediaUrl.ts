interface AppConfig {
  mediaBaseURL?: string
}

let defaultMediaBaseURL = ''

export async function loadMediaBaseURL(): Promise<void> {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 3000)

  try {
    const response = await fetch(`${import.meta.env.BASE_URL}app-config.json`, {
      cache: 'no-store',
      signal: controller.signal,
    })
    if (!response.ok) return

    const appConfig = await response.json() as AppConfig
    defaultMediaBaseURL = String(appConfig.mediaBaseURL || '').trim()
  } catch {
    // VITE_MEDIA_BASE_URL or the current origin remains available as a fallback.
  } finally {
    window.clearTimeout(timeout)
  }
}

function configuredMediaBaseURL() {
  const configured = String(import.meta.env.VITE_MEDIA_BASE_URL || '').trim()
  const value = configured || defaultMediaBaseURL || window.location.origin
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
