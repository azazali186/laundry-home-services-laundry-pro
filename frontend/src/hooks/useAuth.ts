import { useCallback, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { loggedIn, loggedOut, userRefreshed, useAppDispatch, useAppSelector } from '@/store'
import { fetchMe, login as apiLogin, logout as apiLogout } from '@/services/auth.service'
import type { LoginRequest } from '@/services/auth.service'

/** Auth session hook — syncs token/session between API, Redux and router. */
export function useAuth() {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { token, user, hydrated } = useAppSelector((s) => s.auth)

  const loginMutation = useMutation({
    mutationFn: (req: LoginRequest) => apiLogin(req),
    onSuccess: (res) => {
      dispatch(loggedIn({ token: res.access_token, user: res.user }))
      queryClient.clear()
    },
  })

  const logoutMutation = useMutation({
    mutationFn: () => apiLogout(),
    onSettled: () => {
      dispatch(loggedOut())
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  // Refresh the profile once per session to keep roles fresh.
  const meQuery = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: fetchMe,
    enabled: !!token && !user,
    staleTime: 5 * 60_000,
  })

  useEffect(() => {
    if (meQuery.data) {
      dispatch(userRefreshed(meQuery.data))
    }
  }, [meQuery.data, dispatch])

  const logout = useCallback(() => {
    void logoutMutation.mutateAsync()
  }, [logoutMutation])

  const login = useCallback(
    (req: LoginRequest) => loginMutation.mutateAsync(req),
    [loginMutation],
  )

  return {
    token,
    user,
    hydrated,
    isAuthenticated: !!token,
    isLoggingIn: loginMutation.isPending,
    login,
    logout,
  }
}
