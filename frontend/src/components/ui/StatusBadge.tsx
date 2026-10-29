import { clsx } from 'clsx'
import type { MonitorStatus } from '../../types'

interface Props {
  status: MonitorStatus
  className?: string
}

const statusConfig: Record<MonitorStatus, { label: string; classes: string }> = {
  up: { label: 'Up', classes: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' },
  down: { label: 'Down', classes: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' },
  pending: { label: 'Pending', classes: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400' },
  paused: { label: 'Paused', classes: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400' },
}

export function StatusBadge({ status, className }: Props) {
  const config = statusConfig[status] ?? statusConfig.pending
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        config.classes,
        className,
      )}
    >
      <span
        className={clsx('mr-1.5 h-1.5 w-1.5 rounded-full', {
          'bg-green-500': status === 'up',
          'bg-red-500': status === 'down',
          'bg-yellow-500': status === 'pending',
          'bg-gray-400': status === 'paused',
        })}
      />
      {config.label}
    </span>
  )
}
