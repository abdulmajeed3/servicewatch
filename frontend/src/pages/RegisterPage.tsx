import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod/v4'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Activity } from 'lucide-react'
import { register as registerUser } from '../api/auth'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import type { ApiError } from '../api/client'

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
  email: z.email('Invalid email'),
  password: z
    .string()
    .min(8, 'At least 8 characters')
    .regex(/[A-Z]/, 'Must contain an uppercase letter')
    .regex(/[a-z]/, 'Must contain a lowercase letter')
    .regex(/[0-9]/, 'Must contain a digit'),
})
type FormData = z.infer<typeof schema>

export function RegisterPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  const mutation = useMutation({
    mutationFn: ({ email, name, password }: FormData) => registerUser(email, name, password),
    onSuccess: (data) => {
      qc.setQueryData(['me'], data.user)
      navigate('/dashboard', { replace: true })
    },
    onError: (err: ApiError) => {
      setError('root', { message: err.message })
    },
  })

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-950">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center gap-2">
          <Activity className="h-10 w-10 text-blue-600" />
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">ServiceWatch</h1>
          <p className="text-sm text-gray-500">Create your account</p>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-900">
          <form onSubmit={handleSubmit((data) => mutation.mutate(data))} className="space-y-4" noValidate>
            {errors.root && (
              <div className="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
                {errors.root.message}
              </div>
            )}

            <Input label="Name" autoComplete="name" {...register('name')} error={errors.name?.message} />
            <Input label="Email" type="email" autoComplete="email" {...register('email')} error={errors.email?.message} />
            <Input
              label="Password"
              type="password"
              autoComplete="new-password"
              {...register('password')}
              error={errors.password?.message}
              helpText="8+ chars, uppercase, lowercase, digit"
            />

            <Button type="submit" className="w-full" loading={mutation.isPending}>
              Create account
            </Button>
          </form>

          <p className="mt-6 text-center text-sm text-gray-600 dark:text-gray-400">
            Already have an account?{' '}
            <Link to="/login" className="font-medium text-blue-600 hover:underline">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
