import { useCallback, useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Provider } from 'react-redux'
import { RouterProvider } from 'react-router-dom'
import { store, useAppDispatch } from '@/store'
import { loggedOut } from '@/store/auth.slice'
import { setUnauthorizedHandler } from '@/lib/api'
import { ToastProvider } from '@/components/ui/toast'
import { ErrorBoundary } from '@/components/error-boundary'
import { router } from '@/router'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
    mutations: {
      retry: 0,
    },
  },
})

/** Wires the axios 401 handler to the Redux session so a single
 *  global logout keeps the app consistent. */
function SessionBridge() {
  const dispatch = useAppDispatch()

  const handleUnauthorized = useCallback(() => {
    dispatch(loggedOut())
  }, [dispatch])

  useEffect(() => {
    setUnauthorizedHandler(handleUnauthorized)
    return () => setUnauthorizedHandler(null)
  }, [handleUnauthorized])

  return null
}

export default function App() {
  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <SessionBridge />
          <ErrorBoundary>
            <RouterProvider router={router} />
          </ErrorBoundary>
        </ToastProvider>
      </QueryClientProvider>
    </Provider>
  )
}
