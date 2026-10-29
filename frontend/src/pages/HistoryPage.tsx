import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getChecks, getAvailability } from '../api/monitors'
import { Card, CardTitle } from '../components/ui/Card'
import { StatusBadge } from '../components/ui/StatusBadge'
import { Button } from '../components/ui/Button'
import { PageSpinner } from '../components/ui/Spinner'
import { formatDistanceToNow } from '../utils/time'

export function HistoryPage() {
  const { id } = useParams<{ id: string }>()
  const monitorId = Number(id)
  const [page, setPage] = useState(1)
  const [onlyFails, setOnlyFails] = useState(false)

  const { data: avail } = useQuery({
    queryKey: ['availability', monitorId, 24],
    queryFn: () => getAvailability(monitorId, 24),
  })

  const { data: avail7 } = useQuery({
    queryKey: ['availability', monitorId, 168],
    queryFn: () => getAvailability(monitorId, 168),
  })

  const { data, isLoading } = useQuery({
    queryKey: ['checks', monitorId, page, onlyFails],
    queryFn: () => getChecks(monitorId, { page, limit: 50, only_fails: onlyFails }),
    refetchInterval: 30_000,
  })

  if (isLoading) return <PageSpinner />

  const checks = data?.checks ?? []
  const total = data?.total_count ?? 0
  const totalPages = Math.ceil(total / 50)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Check History</h1>
        <Link to={`/monitors/${monitorId}`}>
          <Button variant="secondary" size="sm">← Back</Button>
        </Link>
      </div>

      {/* Availability cards */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <p className="text-xs text-gray-500 uppercase">24h Availability</p>
          <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
            {avail ? `${avail.availability.toFixed(2)}%` : '—'}
          </p>
          <p className="text-xs text-gray-400">{avail ? `${avail.successful}/${avail.total} checks` : ''}</p>
        </Card>
        <Card>
          <p className="text-xs text-gray-500 uppercase">7-Day Availability</p>
          <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
            {avail7 ? `${avail7.availability.toFixed(2)}%` : '—'}
          </p>
          <p className="text-xs text-gray-400">{avail7 ? `${avail7.successful}/${avail7.total} checks` : ''}</p>
        </Card>
      </div>

      {/* Checks table */}
      <Card padding={false}>
        <div className="flex items-center justify-between px-6 py-4">
          <CardTitle>Checks ({total})</CardTitle>
          <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <input
              type="checkbox"
              checked={onlyFails}
              onChange={(e) => { setOnlyFails(e.target.checked); setPage(1) }}
              className="rounded"
            />
            Failures only
          </label>
        </div>
        {checks.length === 0 ? (
          <p className="px-6 pb-6 text-sm text-gray-400">No checks found.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 bg-gray-50 text-xs font-medium uppercase text-gray-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">
                  <th className="px-4 py-3 text-left">Time</th>
                  <th className="px-4 py-3 text-left">Result</th>
                  <th className="px-4 py-3 text-left">Status</th>
                  <th className="px-4 py-3 text-left">Response</th>
                  <th className="px-4 py-3 text-left">Reason</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
                {checks.map((c) => (
                  <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/40">
                    <td className="px-4 py-2 text-gray-500">{formatDistanceToNow(c.checked_at)}</td>
                    <td className="px-4 py-2"><StatusBadge status={c.is_success ? 'up' : 'down'} /></td>
                    <td className="px-4 py-2 text-gray-600 dark:text-gray-400">{c.status_code ?? '—'}</td>
                    <td className="px-4 py-2 text-gray-600 dark:text-gray-400">{c.response_ms != null ? `${c.response_ms}ms` : '—'}</td>
                    <td className="px-4 py-2 text-gray-500 dark:text-gray-400">{c.failure_reason ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-gray-100 px-4 py-3 dark:border-gray-800">
            <p className="text-xs text-gray-500">{total} checks</p>
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
    </div>
  )
}
