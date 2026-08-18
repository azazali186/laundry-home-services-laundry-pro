import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

export function NotFoundPage() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center animate-fade-in">
      <p className="bg-gradient-to-r from-brand-600 to-violet-600 bg-clip-text text-7xl font-bold text-transparent">
        404
      </p>
      <h1 className="mt-4 text-lg font-bold text-surface-900">Page not found</h1>
      <p className="mt-1.5 max-w-sm text-sm text-surface-500">
        The page you're looking for doesn't exist or you don't have access to it.
      </p>
      <Link
        to="/app"
        className="mt-6 inline-flex items-center gap-2 rounded-lg border border-surface-300 bg-white px-4 py-2 text-sm font-medium text-surface-700 shadow-sm transition-all duration-150 hover:bg-surface-100 active:scale-[0.985]"
      >
        <ArrowLeft className="size-4" /> Back to overview
      </Link>
    </div>
  )
}
