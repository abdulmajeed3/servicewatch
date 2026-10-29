import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Search, Edit2, Trash2, PauseCircle, PlayCircle, Eye } from 'lucide-react'
import { listMonitors, pauseMonitor, resumeMonitor, deleteMonitor } from '../api/monitors'
import { Card } from '../components/ui/Card'
import { StatusBadge } from '../components/ui/StatusBadge'
import { Button } from '../components/ui/Button'
import { Select } from '../components/ui/Select'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { PageSpinner } from '../components/ui/Spinner'
import type { Monitor, MonitorStatus } from '../types'
import { formatDistanceToNow, intervalLabel } from '../utils/time'

export function MonitorListPage() {
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<MonitorStatus | ''>('')
  const [page, setPage] = useState(1)
  const [deleteTarget, setDeleteTarget] = useState<Monitor | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['monitors', { search, statusFilter, page }],
    queryFn: () => listMonitors({ search, status: statusFilter, page, limit: 20 }),
    refetchInterval: 30_000,
  })

  const pauseMut = useMutation({
    mutationFn: (id: number) => pauseMonitor(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['monitors'] }),
  })

  const resumeMut = useMutation({
    mutationFn: (id: number) => resumeMonitor(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['monitors'] }),
  })

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteMonitor(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['monitors'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      setDeleteTarget(null)
    },
  })

  if (isLoading) return <PageSpinner />

  const monitors = data?.monitors ?? []
  const total = data?.total_count ?? 0
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Monitors</h1>
        <Link to="/monitors/new">
          <Button size="sm">
            <Plus className="h-4 w-4" />
            New Monitor
          </Button>
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="search"
            placeholder="Search monitors…"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            className="w-full rounded-md border border-gray-300 py-2 pl-9 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
          />
        </div>
        <Select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value as MonitorStatus | ''); setPage(1) }}
          className="w-full sm:w-40"
        >
          <option value="">All statuses</option>
          <option value="up">Up</option>
          <option value="down">Down</option>
          <option value="pending">Pending</option>
          <option value="paused">Paused</option>
        </Select>
      </div>

      {monitors.length === 0 ? (
        <Card className="flex flex-col items-center gap-4 py-16 text-center">
          <p className="text-gray-500">No monitors found.</p>
          <Link to="/monitors/new">
            <Button size="sm">
              <Plus className="h-4 w-4" /> Add one
            </Button>
          </Link>
        </Card>
      ) : (
        <Card padding={false}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 bg-gray-50 text-left text-xs font-medium uppercase text-gray-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">
                  <th className="px-4 py-3">Monitor</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Response</th>
                  <th className="px-4 py-3">Last Check</th>
                  <th className="px-4 py-3">Interval</th>
                  <th className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                {monitors.map((m) => (
                  <tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3">
                      <Link to={`/monitors/${m.id}`} className="font-medium text-gray-900 hover:text-blue-600 dark:text-gray-100">
                        {m.name}
                      </Link>
                      <p className="truncate text-xs text-gray-400 max-w-xs">{m.url}</p>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={m.status} />
                    </td>
                    <td className="px-4 py-3 text-gray-600 dark:text-gray-400">
                      {m.last_response_ms != null ? `${m.last_response_ms}ms` : '—'}
                    </td>
                    <td className="px-4 py-3 text-gray-500 dark:text-gray-400">
                      {m.last_checked_at ? formatDistanceToNow(m.last_checked_at) : '—'}
                    </td>
                    <td className="px-4 py-3 text-gray-500 dark:text-gray-400">
                      {intervalLabel(m.check_interval)}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        <Link to={`/monitors/${m.id}`}>
                          <Button variant="ghost" size="sm" title="View">
                            <Eye className="h-3.5 w-3.5" />
                          </Button>
                        </Link>
                        <Link to={`/monitors/${m.id}/edit`}>
                          <Button variant="ghost" size="sm" title="Edit">
                            <Edit2 className="h-3.5 w-3.5" />
                          </Button>
                        </Link>
                        {m.is_active ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            title="Pause"
                            onClick={() => pauseMut.mutate(m.id)}
                            loading={pauseMut.isPending}
                          >
                            <PauseCircle className="h-3.5 w-3.5" />
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            title="Resume"
                            onClick={() => resumeMut.mutate(m.id)}
                            loading={resumeMut.isPending}
                          >
                            <PlayCircle className="h-3.5 w-3.5 text-green-500" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          title="Delete"
                          onClick={() => setDeleteTarget(m)}
                        >
                          <Trash2 className="h-3.5 w-3.5 text-red-400" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-gray-100 px-4 py-3 dark:border-gray-800">
              <p className="text-xs text-gray-500">{total} monitors</p>
              <div className="flex gap-2">
                <Button variant="secondary" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                  Previous
                </Button>
                <Button variant="secondary" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      )}

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        title="Delete monitor"
        description={`Delete "${deleteTarget?.name}"? This will remove all check history and incidents.`}
        confirmLabel="Delete"
        onConfirm={() => deleteTarget && deleteMut.mutate(deleteTarget.id)}
        onCancel={() => setDeleteTarget(null)}
        loading={deleteMut.isPending}
      />
    </div>
  )
}
