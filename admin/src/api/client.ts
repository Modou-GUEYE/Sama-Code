import type { TokenPair } from './types'

/** Erreur normalisée de l'API : {"error": {"code", "message", "details"}}. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details: Record<string, unknown>

  constructor(status: number, code: string, message: string, details: Record<string, unknown> = {}) {
    super(message)
    this.status = status
    this.code = code
    this.details = details
  }
}

const ACCESS = 'samacode.admin.access'
const REFRESH = 'samacode.admin.refresh'

function read(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export const session = {
  get access() {
    return read(ACCESS)
  },
  get refresh() {
    return read(REFRESH)
  },
  save(pair: TokenPair) {
    try {
      localStorage.setItem(ACCESS, pair.access_token)
      localStorage.setItem(REFRESH, pair.refresh_token)
    } catch {
      /* stockage indisponible : la session ne survivra pas au rechargement */
    }
  },
  clear() {
    try {
      localStorage.removeItem(ACCESS)
      localStorage.removeItem(REFRESH)
    } catch {
      /* rien à faire */
    }
  },
}

/** Appelé quand la session ne peut plus être renouvelée. */
let onExpired: () => void = () => {}
export function onSessionExpired(fn: () => void) {
  onExpired = fn
}

let refreshing: Promise<boolean> | null = null

// Un seul renouvellement à la fois : le jeton de rafraîchissement est à usage unique.
function refresh(): Promise<boolean> {
  refreshing ??= (async () => {
    const token = session.refresh
    if (!token) return false
    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: token }),
      })
      if (!res.ok) return false
      session.save(await res.json())
      return true
    } catch {
      return false
    }
  })().finally(() => {
    refreshing = null
  })
  return refreshing
}

async function request<T>(method: string, path: string, body?: unknown, retried = false): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = session.access
  if (token) headers.Authorization = `Bearer ${token}`
  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  let res: Response
  try {
    res = await fetch(`/api/v1${path}`, { method, headers, body: payload })
  } catch {
    throw new ApiError(0, 'network_error', 'Connexion au serveur impossible.')
  }

  if (res.status === 401 && !retried && !path.startsWith('/auth/') && (await refresh())) {
    return request<T>(method, path, body, true)
  }
  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/')) {
      session.clear()
      onExpired()
    }
    const e = data?.error ?? {}
    throw new ApiError(res.status, e.code ?? 'unknown_error', e.message ?? 'Erreur inattendue.', e.details ?? {})
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
}

/** Message affichable, avec le détail utile quand l'API en fournit un. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    const field = err.details.field ?? err.details.missing
    return field ? `${err.message} (${String(field)})` : err.message
  }
  return 'Erreur inattendue.'
}
