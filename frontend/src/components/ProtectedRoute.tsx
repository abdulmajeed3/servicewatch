import { Navigate, useLocation } from 'react-router-dom'
import { useCurrentUser } from '../hooks/useAuth'
import { PageSpinner } from './ui/Spinner'

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { data: user, isLoading, isError } = useCurrentUser()
  const location = useLocation()

  if (isLoading) return <PageSpinner />
  if (isError || !user) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }
  return <>{children}</>
}
