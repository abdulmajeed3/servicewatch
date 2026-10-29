import { api, setToken, clearToken } from './client'
import type { TokenResponse, User } from '../types'

export async function register(email: string, name: string, password: string): Promise<TokenResponse> {
  const resp = await api.post<TokenResponse>('/auth/register', { email, name, password })
  setToken(resp.token)
  return resp
}

export async function login(email: string, password: string): Promise<TokenResponse> {
  const resp = await api.post<TokenResponse>('/auth/login', { email, password })
  setToken(resp.token)
  return resp
}

export function logout(): void {
  clearToken()
}

export function getMe(): Promise<User> {
  return api.get<User>('/auth/me')
}

export function updateProfile(name: string): Promise<User> {
  return api.patch<User>('/auth/me', { name })
}

export function changePassword(current_password: string, new_password: string): Promise<void> {
  return api.post<void>('/auth/change-password', { current_password, new_password })
}
