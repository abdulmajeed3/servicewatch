import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod/v4'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createMonitor, getMonitor, updateMonitor } from '../api/monitors'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Button } from '../components/ui/Button'
import { Card, CardHeader, CardTitle } from '../components/ui/Card'
import { PageSpinner } from '../components/ui/Spinner'

const INTERVALS = [
  { value: 60, label: '1 minute' },
  { value: 300, label: '5 minutes' },
  { value: 600, label: '10 minutes' },
  { value: 1800, label: '30 minutes' },
  { value: 3600, label: '1 hour' },
]

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(200),
  url: z.url('Must be a valid URL').refine(
    (u) => u.startsWith('http://') || u.startsWith('https://'),
    'Must use http or https',
  ),
  check_interval: z.number().int().min(60),
  request_timeout: z.number().int().min(5).max(120),
  expected_status: z.number().int().min(0).max(599),
})
type FormData = z.infer<typeof schema>

export function MonitorFormPage() {
  const { id } = useParams<{ id?: string }>()
  const isEdit = Boolean(id)
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: existing, isLoading } = useQuery({
    queryKey: ['monitor', id],
    queryFn: () => getMonitor(Number(id)),
    enabled: isEdit,
  })

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
    setError,
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      check_interval: 300,
      request_timeout: 30,
      expected_status: 0,
    },
  })

  useEffect(() => {
    if (existing) {
      reset({
        name: existing.name,
        url: existing.url,
        check_interval: existing.check_interval,
        request_timeout: existing.request_timeout,
        expected_status: existing.expected_status,
      })
    }
  }, [existing, reset])

  const mutation = useMutation({
    mutationFn: (data: FormData) =>
      isEdit ? updateMonitor(Number(id), data) : createMonitor(data),
    onSuccess: (m) => {
      qc.invalidateQueries({ queryKey: ['monitors'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      navigate(`/monitors/${m.id}`)
    },
    onError: (err: Error) => {
      setError('root', { message: err.message })
    },
  })

  if (isLoading) return <PageSpinner />

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
        {isEdit ? 'Edit Monitor' : 'New Monitor'}
      </h1>

      <Card>
        <CardHeader>
          <CardTitle>{isEdit ? 'Update configuration' : 'Monitor details'}</CardTitle>
        </CardHeader>

        <form onSubmit={handleSubmit((d) => mutation.mutate(d))} className="space-y-4" noValidate>
          {errors.root && (
            <div className="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
              {errors.root.message}
            </div>
          )}

          <Input
            label="Name"
            placeholder="My API"
            {...register('name')}
            error={errors.name?.message}
          />

          <Input
            label="URL"
            type="url"
            placeholder="https://api.example.com/health"
            {...register('url')}
            error={errors.url?.message}
            helpText="Must use http or https. Credentials in the URL are not supported."
          />

          <Select
            label="Check interval"
            {...register('check_interval', { valueAsNumber: true })}
            error={errors.check_interval?.message}
          >
            {INTERVALS.map((i) => (
              <option key={i.value} value={i.value}>{i.label}</option>
            ))}
          </Select>

          <Input
            label="Request timeout (seconds)"
            type="number"
            min={5}
            max={120}
            {...register('request_timeout', { valueAsNumber: true })}
            error={errors.request_timeout?.message}
            helpText="5–120 seconds"
          />

          <Input
            label="Expected HTTP status"
            type="number"
            min={0}
            max={599}
            {...register('expected_status', { valueAsNumber: true })}
            error={errors.expected_status?.message}
            helpText="Set to 0 to accept any 2xx response"
          />

          <div className="flex gap-3 pt-2">
            <Button type="submit" loading={mutation.isPending}>
              {isEdit ? 'Save changes' : 'Create monitor'}
            </Button>
            <Button type="button" variant="secondary" onClick={() => navigate(-1)}>
              Cancel
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
