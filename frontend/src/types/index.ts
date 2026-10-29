// Core domain types mirroring the backend API shapes.

export type MonitorStatus = 'up' | 'down' | 'pending' | 'paused'

export interface User {
  id: number
  email: string
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface TokenResponse {
  token: string
  expires_at: string
  user: User
}

export interface Monitor {
  id: number
  user_id: number
  name: string
  url: string
  check_interval: number
  request_timeout: number
  expected_status: number
  status: MonitorStatus
  is_active: boolean
  next_check_at: string
  last_checked_at: string | null
  last_status_code: number | null
  last_response_ms: number | null
  consecutive_failures: number
  created_at: string
  updated_at: string
}

export interface MonitorListResult {
  monitors: Monitor[]
  total_count: number
  page: number
  limit: number
}

export interface CreateMonitorInput {
  name: string
  url: string
  check_interval: number
  request_timeout: number
  expected_status: number
}

export interface DashboardStats {
  total: number
  up: number
  down: number
  paused: number
  pending: number
}

export interface MonitorCheck {
  id: number
  monitor_id: number
  user_id: number
  checked_at: string
  is_success: boolean
  status_code: number | null
  response_ms: number | null
  failure_reason: string | null
  created_at: string
}

export interface CheckListResult {
  checks: MonitorCheck[]
  total_count: number
  page: number
  limit: number
}

export interface AvailabilityStat {
  window_hours: number
  total: number
  successful: number
  availability: number
}

export interface ResponseTimePoint {
  checked_at: string
  response_ms: number
  is_success: boolean
}

export interface Incident {
  id: number
  monitor_id: number
  user_id: number
  started_at: string
  resolved_at: string | null
  failure_count: number
  last_failure_at: string
  created_at: string
  updated_at: string
}

export interface IncidentListResult {
  incidents: Incident[]
  total_count: number
  page: number
  limit: number
}

export interface DashboardResponse {
  stats: DashboardStats
  recent_failures: MonitorCheck[]
  recent_incidents: Incident[]
}

export interface APIError {
  error: string
  fields?: Record<string, string>
}

export interface ListFilter {
  page?: number
  limit?: number
  search?: string
  status?: MonitorStatus | ''
}
