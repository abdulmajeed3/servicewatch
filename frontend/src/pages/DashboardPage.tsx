import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Activity, TrendingUp, TrendingDown, PauseCircle, AlertCircle, Plus, Gauge } from 'lucide-react'
import { getDashboard } from '../api/monitors'
import { Card, CardHeader, CardTitle } from '../components/ui/Card'
import { StatusBadge } from '../components/ui/StatusBadge'
import { Button } from '../components/ui/Button'
import { PageSpinner } from '../components/ui/Spinner'
import { formatDistanceToNow } from '../utils/time'

function StatCard({
  label,
  value,
  icon: Icon,
  color,
}: {
  label: string
  value: number
  icon: React.ComponentType<{ className?: string }>
  color: string
}) {
  return (
    <Card className="flex items-center gap-4">
      <div className={`rounded-lg p-3 ${color}`}>
        <Icon className="h-5 w-5 text-white" />
      </div>
      <div>
        <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
        <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
      </div>
    </Card>
  )
}

export function DashboardPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['dashboard'],
    queryFn: getDashboard,
    refetchInterval: 30_000,
  })

  if (isLoading) return <PageSpinner />

  if (isError || !data) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <AlertCircle className="h-10 w-10 text-red-400" />
        <p className="text-gray-600 dark:text-gray-400">Failed to load dashboard.</p>
      </div>
    )
  }

  const { stats, overall_availability_24h, recent_failures, recent_incidents } = data

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <Link to="/monitors/new">
          <Button size="sm">
            <Plus className="h-4 w-4" />
            Add Monitor
          </Button>
        </Link>
      </div>

      {/* Stats grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total Monitors" value={stats.total} icon={Activity} color="bg-blue-500" />
        <StatCard label="Up" value={stats.up} icon={TrendingUp} color="bg-green-500" />
        <StatCard label="Down" value={stats.down} icon={TrendingDown} color="bg-red-500" />
        <StatCard label="Paused" value={stats.paused} icon={PauseCircle} color="bg-gray-400" />
      </div>

      {/* Overall availability banner */}
      {stats.total > 0 && (
        <Card className="flex items-center gap-4">
          <div className="rounded-lg bg-indigo-500 p-3">
            <Gauge className="h-5 w-5 text-white" />
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900 dark:text-white">
              {overall_availability_24h.toFixed(2)}%
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">Overall availability (last 24h)</p>
          </div>
        </Card>
      )}

      {/* Empty state */}
      {stats.total === 0 && (
        <Card className="flex flex-col items-center gap-4 py-16 text-center">
          <Activity className="h-12 w-12 text-gray-300 dark:text-gray-600" />
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">No monitors yet</h2>
            <p className="mt-1 text-sm text-gray-500">Add your first endpoint to start monitoring.</p>
          </div>
          <Link to="/monitors/new">
            <Button>
              <Plus className="h-4 w-4" />
              Create monitor
            </Button>
          </Link>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent failures */}
        {recent_failures && recent_failures.length > 0 && (
          <Card padding={false}>
            <CardHeader className="px-6 pt-6">
              <CardTitle>Recent Failures</CardTitle>
            </CardHeader>
            <div className="divide-y divide-gray-100 dark:divide-gray-800">
              {recent_failures.map((check) => (
                <Link
                  key={check.id}
                  to={`/monitors/${check.monitor_id}`}
                  className="flex items-center justify-between px-6 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/40"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {check.monitor_name ?? `Monitor #${check.monitor_id}`}
                    </p>
                    <p className="text-xs text-red-600 dark:text-red-400">
                      {check.failure_reason ?? `HTTP ${check.status_code}`}
                    </p>
                  </div>
                  <span className="text-xs text-gray-400">{formatDistanceToNow(check.checked_at)}</span>
                </Link>
              ))}
            </div>
          </Card>
        )}

        {/* Recent incidents */}
        {recent_incidents && recent_incidents.length > 0 && (
          <Card padding={false}>
            <CardHeader className="px-6 pt-6">
              <CardTitle>Recent Incidents</CardTitle>
            </CardHeader>
            <div className="divide-y divide-gray-100 dark:divide-gray-800">
              {recent_incidents.map((inc) => (
                <Link
                  key={inc.id}
                  to={`/monitors/${inc.monitor_id}`}
                  className="flex items-center justify-between px-6 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/40"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {inc.monitor_name ?? `Monitor #${inc.monitor_id}`}
                    </p>
                    <p className="text-xs text-gray-500">
                      {inc.failure_count} failure{inc.failure_count !== 1 ? 's' : ''}
                    </p>
                  </div>
                  <div className="text-right">
                    <StatusBadge status={inc.resolved_at ? 'up' : 'down'} />
                    <p className="mt-1 text-xs text-gray-400">{formatDistanceToNow(inc.started_at)}</p>
                  </div>
                </Link>
              ))}
            </div>
          </Card>
        )}
      </div>
    </div>
  )
}
