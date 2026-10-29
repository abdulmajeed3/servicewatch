import type { APIError } from '../types'

const BASE_URL = '/api/v1'

export class ApiError extends Error {
  status: number
  body: APIError

  constructor(status: number, body: APIError) {
    super(body.error || `HTTP ${status}`)
    this.status = status
    this.body = body
  }
}

function getToken(): string | null {
  try {
    return localStorage.getItem('sw_token')
  } catch {
    return null
  }
}

export function setToken(token: string): void {
  try {
    localStorage.setItem('sw_token', token)
  } catch {
    // ignore — storage may be unavailable
  }
}

export function clearToken(): void {
  try {
    localStorage.removeItem('sw_token')
  } catch {
    // ignore
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    let errorBody: APIError = { error: `HTTP ${res.status}` }
    try {
      const json = await res.json()
      if (json?.error) errorBody = json
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, errorBody)
  }

  if (res.status === 204) {
    return undefined as T
  }

  const json = await res.json()
  return json?.data !== undefined ? json.data : json
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
}
