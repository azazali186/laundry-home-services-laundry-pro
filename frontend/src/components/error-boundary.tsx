import { Component, type ErrorInfo, type ReactNode } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  message: string
}

/**
 * Top-level error boundary. A render error anywhere in the app must never
 * produce a white screen: users get a friendly recovery card instead.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false, message: '' }

  static getDerivedStateFromError(error: unknown): ErrorBoundaryState {
    return {
      hasError: true,
      message: error instanceof Error ? error.message : 'Unexpected application error',
    }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Error reporting only — the boundary has already handled the UI.
    console.error('Uncaught render error:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-dvh flex-col items-center justify-center bg-surface-50 px-6 text-center">
          <div className="flex size-14 items-center justify-center rounded-2xl bg-red-50 text-red-600">
            <AlertTriangle className="size-7" aria-hidden />
          </div>
          <h1 className="mt-5 font-display text-2xl font-bold tracking-tight text-surface-900">
            Something went wrong
          </h1>
          <p className="mt-2 max-w-md text-sm leading-relaxed text-surface-500">
            An unexpected error occurred while rendering this page. Your data is safe — reload to
            continue.
          </p>
          {/* Internal details only in development — never leak to end users. */}
          {import.meta.env.DEV && this.state.message && (
            <p className="mt-6 max-w-md truncate rounded-lg bg-surface-100 px-3 py-1.5 font-mono text-[11px] text-surface-400">
              {this.state.message}
            </p>
          )}
          <button
            onClick={() => window.location.reload()}
            className="mt-6 inline-flex items-center gap-2 rounded-xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition-all duration-200 hover:bg-brand-700 active:scale-[0.98]"
          >
            <RefreshCw className="size-4" aria-hidden /> Reload page
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
