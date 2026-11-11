import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Edit2, Trash2, PauseCircle, PlayCircle, AlertCircle, Clock, Wifi, RefreshCw } from 'lucide-react'
import {
  getMonitor,
  deleteMonitor,
  pauseMonitor,
  resumeMonitor,
  checkNowMonitor,
  getAvailability,
  getResponseTimes,
  getChecks,
} from '../api/monitors'
import { Card, CardHeader, CardTitle } from '../components/ui/Card'
import { StatusBadge } from '../components/ui/StatusBadge'
import { Button } from '../components/ui/Button'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { PageSpinner } from '../components/ui/Spinner'
import { ResponseTimeChart } from '../components/ResponseTimeChart'
import { formatDate, formatDistanceToNow, intervalLabel } from '../utils/time'

export function MonitorDetailPage() {
  const { id } = useParams<{ id: string }>()
  const monitorId = Number(id)
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [showDelete, setShowDelete] = useState(false)

  const { data: monitor, isLoading } = useQuery({
    queryKey: ['monitor', id],
    queryFn: () => getMonitor(monitorId),
    refetchInterval: 30_000,
  })

  const { data: availability } = useQuery({
    queryKey: ['availability', monitorId, 24],
    queryFn: () => getAvailability(monitorId, 24),
    enabled: Boolean(monitor),
  })

  const { data: timeSeries } = useQuery({
    queryKey: ['response-times', monitorId, 24],
    queryFn: () => getResponseTimes(monitorId, 24),
    enabled: Boolean(monitor),
  })

  const { data: checks } = useQuery({
    queryKey: ['checks', monitorId, 1],
    queryFn: () => getChecks(monitorId, { page: 1, limit: 20 }),
    enabled: Boolean(monitor),
    refetchInterval: 30_000,
  })

  const pauseMut = useMutation({
    mutationFn: () => pauseMonitor(monitorId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['monitor', id] }),
  })

  const resumeMut = useMutation({
    mutationFn: () => resumeMonitor(monitorId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['monitor', id] }),
  })

  const checkNowMut = useMutation({
    mutationFn: () => checkNowMonitor(monitorId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['monitor', id] })
      qc.invalidateQueries({ queryKey: ['checks', monitorId] })
      qc.invalidateQueries({ queryKey: ['availability', monitorId] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })

  const deleteMut = useMutation({
    mutationFn: () => deleteMonitor(monitorId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['monitors'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      navigate('/monitors')
    },
  })

  if (isLoading) return <PageSpinner />
  if (!monitor) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <AlertCircle className="h-10 w-10 text-red-400" />
        <p className="text-gray-600 dark:text-gray-400">Monitor not found.</p>
        <Link to="/monitors"><Button variant="secondary">Back to monitors</Button></Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <StatusBadge status={monitor.status} />
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{monitor.name}</h1>
        </div>
        <div className="flex flex-wrap gap-2">
          {monitor.is_active && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => checkNowMut.mutate()}
              loading={checkNowMut.isPending}
              title="Run an immediate check without changing the schedule"
            >
              <RefreshCw className="h-4 w-4" /> Check Now
            </Button>
          )}
          {monitor.is_active ? (
            <Button variant="secondary" size="sm" onClick={() => pauseMut.mutate()} loading={pauseMut.isPending}>
              <PauseCircle className="h-4 w-4" /> Pause
            </Button>
          ) : (
            <Button variant="secondary" size="sm" onClick={() => resumeMut.mutate()} loading={resumeMut.isPending}>
              <PlayCircle className="h-4 w-4" /> Resume
            </Button>
          )}
          <Link to={`/monitors/${monitorId}/edit`}>
            <Button variant="secondary" size="sm"><Edit2 className="h-4 w-4" /> Edit</Button>
          </Link>
          <Button variant="danger" size="sm" onClick={() => setShowDelete(true)}>
            <Trash2 className="h-4 w-4" /> Delete
          </Button>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="flex items-center gap-3">
          <Wifi className="h-5 w-5 text-blue-500 shrink-0" />
          <div>
            <p className="text-xs text-gray-500">URL</p>
            <p className="text-sm font-medium truncate text-gray-900 dark:text-white max-w-[200px]">{monitor.url}</p>
          </div>
        </Card>
        <Card className="flex items-center gap-3">
          <Clock className="h-5 w-5 text-purple-500 shrink-0" />
          <div>
            <p className="text-xs text-gray-500">Interval</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white">{intervalLabel(monitor.check_interval)}</p>
          </div>
        </Card>
        <Card className="flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-orange-500 shrink-0" />
          <div>
            <p className="text-xs text-gray-500">24h Availability</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white">
              {availability ? `${availability.availability.toFixed(1)}%` : '—'}
            </p>
          </div>
        </Card>
      </div>

      {/* Response time chart */}
      {timeSeries && timeSeries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Response Time (last 24h)</CardTitle>
          </CardHeader>
          <ResponseTimeChart data={timeSeries} />
        </Card>
      )}

      {/* Recent checks */}
      {checks && checks.checks.length > 0 && (
        <Card padding={false}>
          <CardHeader className="px-6 pt-6">
            <CardTitle>Recent Checks</CardTitle>
          </CardHeader>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium uppercase text-gray-500 dark:border-gray-700 dark:bg-gray-800">
                  <th className="px-4 py-3">Time</th>
                  <th className="px-4 py-3">Result</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Response</th>
                  <th className="px-4 py-3">Reason</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                {checks.checks.map((c) => (
                  <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/40">
                    <td className="px-4 py-2 text-gray-500">{formatDistanceToNow(c.checked_at)}</td>
                    <td className="px-4 py-2">
                      <StatusBadge status={c.is_success ? 'up' : 'down'} />
                    </td>
                    <td className="px-4 py-2 text-gray-600 dark:text-gray-400">{c.status_code ?? '—'}</td>
                    <td className="px-4 py-2 text-gray-600 dark:text-gray-400">
                      {c.response_ms != null ? `${c.response_ms}ms` : '—'}
                    </td>
                    <td className="px-4 py-2 text-gray-500 dark:text-gray-400">{c.failure_reason ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-800">
            <Link to={`/monitors/${monitorId}/history`} className="text-sm text-blue-600 hover:underline">
              View full history →
            </Link>
          </div>
        </Card>
      )}

      {/* Notes */}
      {monitor.notes && (
        <Card>
          <CardHeader><CardTitle>Notes</CardTitle></CardHeader>
          <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{monitor.notes}</p>
        </Card>
      )}

      {/* Config */}
      <Card>
        <CardHeader><CardTitle>Configuration</CardTitle></CardHeader>
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          {[
            ['Created', formatDate(monitor.created_at)],
            ['Updated', formatDate(monitor.updated_at)],
            ['Timeout', `${monitor.request_timeout}s`],
            ['Expected status', monitor.expected_status === 0 ? 'Any 2xx' : String(monitor.expected_status)],
            ['Consecutive failures', String(monitor.consecutive_failures)],
            ['Last status code', monitor.last_status_code != null ? String(monitor.last_status_code) : '—'],
          ].map(([label, value]) => (
            <div key={label}>
              <dt className="text-gray-500 dark:text-gray-400">{label}</dt>
              <dd className="font-medium text-gray-900 dark:text-white">{value}</dd>
            </div>
          ))}
        </dl>
      </Card>

      <ConfirmDialog
        isOpen={showDelete}
        title="Delete monitor"
        description={`Permanently delete "${monitor.name}" and all its history? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={() => deleteMut.mutate()}
        onCancel={() => setShowDelete(false)}
        loading={deleteMut.isPending}
      />
    </div>
  )
}
