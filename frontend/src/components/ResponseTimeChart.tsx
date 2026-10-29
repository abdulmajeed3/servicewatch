import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts'
import type { ResponseTimePoint } from '../types'

interface Props {
  data: ResponseTimePoint[]
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

export function ResponseTimeChart({ data }: Props) {
  if (!data || data.length === 0) {
    return <p className="py-8 text-center text-sm text-gray-400">No data yet.</p>
  }

  const chartData = data.map((p) => ({
    time: formatTime(p.checked_at),
    ms: p.response_ms,
    success: p.is_success,
  }))

  return (
    <ResponsiveContainer width="100%" height={200}>
      <LineChart data={chartData} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis dataKey="time" tick={{ fontSize: 11 }} tickLine={false} />
        <YAxis
          tick={{ fontSize: 11 }}
          tickLine={false}
          tickFormatter={(v) => `${v}ms`}
          width={50}
        />
        <Tooltip
          formatter={(value) => [`${value}ms`, 'Response time']}
          contentStyle={{ fontSize: 12, borderRadius: 6 }}
        />
        <Line
          type="monotone"
          dataKey="ms"
          stroke="#3b82f6"
          strokeWidth={2}
          dot={false}
          activeDot={{ r: 4 }}
        />
      </LineChart>
    </ResponsiveContainer>
  )
}
