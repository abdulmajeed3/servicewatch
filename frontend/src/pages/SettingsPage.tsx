import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod/v4'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { updateProfile, changePassword } from '../api/auth'
import { Card, CardHeader, CardTitle } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useCurrentUser } from '../hooks/useAuth'

const profileSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
})
type ProfileData = z.infer<typeof profileSchema>

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Required'),
  new_password: z
    .string()
    .min(8, 'At least 8 characters')
    .regex(/[A-Z]/, 'Uppercase letter required')
    .regex(/[a-z]/, 'Lowercase letter required')
    .regex(/[0-9]/, 'Digit required'),
})
type PasswordData = z.infer<typeof passwordSchema>

export function SettingsPage() {
  const qc = useQueryClient()
  const { data: user } = useCurrentUser()

  const profileForm = useForm<ProfileData>({
    resolver: zodResolver(profileSchema),
    values: { name: user?.name ?? '' },
  })

  const passwordForm = useForm<PasswordData>({ resolver: zodResolver(passwordSchema) })

  const profileMut = useMutation({
    mutationFn: ({ name }: ProfileData) => updateProfile(name),
    onSuccess: (updated) => {
      qc.setQueryData(['me'], updated)
      profileForm.reset({ name: updated.name })
    },
    onError: (err: Error) => profileForm.setError('root', { message: err.message }),
  })

  const passwordMut = useMutation({
    mutationFn: ({ current_password, new_password }: PasswordData) =>
      changePassword(current_password, new_password),
    onSuccess: () => {
      passwordForm.reset()
      passwordForm.setError('root', { message: '' })
    },
    onError: (err: Error) => passwordForm.setError('root', { message: err.message }),
  })

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>

      {/* Profile */}
      <Card>
        <CardHeader><CardTitle>Profile</CardTitle></CardHeader>
        <form onSubmit={profileForm.handleSubmit((d) => profileMut.mutate(d))} className="space-y-4" noValidate>
          {profileForm.formState.errors.root?.message && (
            <p className="text-sm text-red-600">{profileForm.formState.errors.root.message}</p>
          )}
          {profileMut.isSuccess && (
            <p className="text-sm text-green-600">Profile updated.</p>
          )}
          <p className="text-sm text-gray-500">{user?.email}</p>
          <Input
            label="Name"
            {...profileForm.register('name')}
            error={profileForm.formState.errors.name?.message}
          />
          <Button type="submit" loading={profileMut.isPending}>Save</Button>
        </form>
      </Card>

      {/* Change password */}
      <Card>
        <CardHeader><CardTitle>Change Password</CardTitle></CardHeader>
        <form onSubmit={passwordForm.handleSubmit((d) => passwordMut.mutate(d))} className="space-y-4" noValidate>
          {passwordForm.formState.errors.root?.message && (
            <p className="text-sm text-red-600">{passwordForm.formState.errors.root.message}</p>
          )}
          {passwordMut.isSuccess && (
            <p className="text-sm text-green-600">Password changed.</p>
          )}
          <Input
            label="Current password"
            type="password"
            {...passwordForm.register('current_password')}
            error={passwordForm.formState.errors.current_password?.message}
          />
          <Input
            label="New password"
            type="password"
            {...passwordForm.register('new_password')}
            error={passwordForm.formState.errors.new_password?.message}
            helpText="8+ chars, uppercase, lowercase, digit"
          />
          <Button type="submit" loading={passwordMut.isPending}>Update password</Button>
        </form>
      </Card>
    </div>
  )
}
