import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getMe, logout as apiLogout } from '../api/auth'

export function useCurrentUser() {
  return useQuery({
    queryKey: ['me'],
    queryFn: getMe,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return () => {
    apiLogout()
    qc.clear()
  }
}
