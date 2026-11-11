import { api } from './client'
import type {
  Monitor,
  MonitorListResult,
  CreateMonitorInput,
  DashboardResponse,
  CheckListResult,
  CheckResult,
  AvailabilityStat,
  ResponseTimePoint,
  IncidentListResult,
  ListFilter,
} from '../types'

export function getDashboard(): Promise<DashboardResponse> {
  return api.get<DashboardResponse>('/dashboard')
}

export function listMonitors(filter: ListFilter = {}): Promise<MonitorListResult> {
  const params = new URLSearchParams()
  if (filter.page) params.set('page', String(filter.page))
  if (filter.limit) params.set('limit', String(filter.limit))
  if (filter.search) params.set('search', filter.search)
  if (filter.status) params.set('status', filter.status)
  const qs = params.toString()
  return api.get<MonitorListResult>(`/monitors${qs ? '?' + qs : ''}`)
}

export function getMonitor(id: number): Promise<Monitor> {
  return api.get<Monitor>(`/monitors/${id}`)
}

export function createMonitor(input: CreateMonitorInput): Promise<Monitor> {
  return api.post<Monitor>('/monitors', input)
}

export function updateMonitor(id: number, input: CreateMonitorInput): Promise<Monitor> {
  return api.put<Monitor>(`/monitors/${id}`, input)
}

export function pauseMonitor(id: number): Promise<Monitor> {
  return api.post<Monitor>(`/monitors/${id}/pause`)
}

export function resumeMonitor(id: number): Promise<Monitor> {
  return api.post<Monitor>(`/monitors/${id}/resume`)
}

export function checkNowMonitor(id: number): Promise<CheckResult> {
  return api.post<CheckResult>(`/monitors/${id}/check`)
}

export function deleteMonitor(id: number): Promise<void> {
  return api.delete<void>(`/monitors/${id}`)
}

export function getChecks(
  monitorId: number,
  filter: { page?: number; limit?: number; only_fails?: boolean } = {},
): Promise<CheckListResult> {
  const params = new URLSearchParams()
  if (filter.page) params.set('page', String(filter.page))
  if (filter.limit) params.set('limit', String(filter.limit))
  if (filter.only_fails) params.set('only_fails', 'true')
  const qs = params.toString()
  return api.get<CheckListResult>(`/monitors/${monitorId}/checks${qs ? '?' + qs : ''}`)
}

export function getAvailability(monitorId: number, windowHours = 24): Promise<AvailabilityStat> {
  return api.get<AvailabilityStat>(`/monitors/${monitorId}/availability?window_hours=${windowHours}`)
}

export function getResponseTimes(monitorId: number, windowHours = 24): Promise<ResponseTimePoint[]> {
  return api.get<ResponseTimePoint[]>(`/monitors/${monitorId}/response-times?window_hours=${windowHours}`)
}

export function getIncidents(
  monitorId: number,
  filter: { page?: number; limit?: number; open?: boolean } = {},
): Promise<IncidentListResult> {
  const params = new URLSearchParams()
  if (filter.page) params.set('page', String(filter.page))
  if (filter.limit) params.set('limit', String(filter.limit))
  if (filter.open !== undefined) params.set('open', String(filter.open))
  const qs = params.toString()
  return api.get<IncidentListResult>(`/monitors/${monitorId}/incidents${qs ? '?' + qs : ''}`)
}
