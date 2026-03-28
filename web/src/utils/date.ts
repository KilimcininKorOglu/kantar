// Timezone-aware date formatting utilities.
// Reads the user's timezone from localStorage (set by AuthContext on login).

const TIMEZONE_KEY = 'kantar_timezone'

export function getTimezone(): string {
  return localStorage.getItem(TIMEZONE_KEY) || Intl.DateTimeFormat().resolvedOptions().timeZone
}

export function setTimezone(tz: string): void {
  localStorage.setItem(TIMEZONE_KEY, tz)
}

export function formatDateTime(isoString: string): string {
  if (!isoString) return '—'
  try {
    return new Intl.DateTimeFormat(undefined, {
      timeZone: getTimezone(),
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(new Date(isoString))
  } catch {
    return new Date(isoString).toLocaleString()
  }
}

export function formatDate(isoString: string): string {
  if (!isoString) return '—'
  try {
    return new Intl.DateTimeFormat(undefined, {
      timeZone: getTimezone(),
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(new Date(isoString))
  } catch {
    return new Date(isoString).toLocaleDateString()
  }
}

export function getTimezoneList(): string[] {
  try {
    return Intl.supportedValuesOf('timeZone')
  } catch {
    // Fallback for older browsers
    return [
      'UTC',
      'Europe/Istanbul',
      'Europe/London',
      'Europe/Berlin',
      'Europe/Paris',
      'Europe/Moscow',
      'America/New_York',
      'America/Chicago',
      'America/Denver',
      'America/Los_Angeles',
      'America/Sao_Paulo',
      'Asia/Tokyo',
      'Asia/Shanghai',
      'Asia/Kolkata',
      'Asia/Dubai',
      'Australia/Sydney',
      'Pacific/Auckland',
    ]
  }
}
