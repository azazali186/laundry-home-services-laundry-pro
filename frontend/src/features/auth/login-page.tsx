import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Eye, EyeOff, Lock, Mail, ShieldCheck, Sparkles, Zap } from 'lucide-react'
import { errorMessage } from '@/lib/api'
import { useAuth } from '@/hooks/useAuth'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/form'
import { useToast } from '@/components/ui/toast'

export function LoginPage() {
  const { login, isLoggingIn } = useAuth()
  const toast = useToast()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/app'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const canSubmit = email.trim() !== '' && password !== '' && !isLoggingIn

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await login({ email: email.trim(), password })
      toast.toast({ title: 'Welcome back', description: 'Signed in successfully.', variant: 'success' })
      navigate(from, { replace: true })
    } catch (err) {
      setError(errorMessage(err, 'Invalid credentials'))
    }
  }

  return (
    <div className="flex min-h-dvh">
      {/* Brand panel */}
      <div className="relative hidden w-1/2 overflow-hidden bg-surface-950 lg:block">
        <div className="absolute inset-0 bg-gradient-to-br from-brand-700 via-brand-900 to-surface-950" />
        <div className="absolute -left-24 -top-24 size-96 rounded-full bg-brand-500/25 blur-3xl" />
        <div className="absolute -bottom-32 right-0 size-96 rounded-full bg-violet-500/20 blur-3xl" />
        <div className="relative flex h-full flex-col justify-between p-12">
          <div className="flex items-center gap-3">
            <div className="flex size-10 items-center justify-center rounded-2xl bg-white/10 text-white backdrop-blur">
              <Sparkles className="size-5" />
            </div>
            <div>
              <p className="text-lg font-bold text-white">LaundryPro</p>
              <p className="text-[10px] font-medium uppercase tracking-[0.2em] text-white/50">
                Operations Console
              </p>
            </div>
          </div>

          <div className="max-w-md">
            <h1 className="text-4xl font-bold leading-tight tracking-tight text-white">
              Run your laundry business with clarity.
            </h1>
            <p className="mt-4 text-sm leading-relaxed text-white/60">
              Bookings, orders, payments and inventory — one calm, fast console for your entire
              operation.
            </p>
            <ul className="mt-8 space-y-3">
              {[
                { icon: Zap, text: 'Real-time operational pipeline tracking' },
                { icon: ShieldCheck, text: 'Role-based access control built-in' },
                { icon: Sparkles, text: 'Audit trails for every action' },
              ].map((f) => (
                <li key={f.text} className="flex items-center gap-3 text-sm text-white/80">
                  <span className="flex size-7 items-center justify-center rounded-lg bg-white/10">
                    <f.icon className="size-3.5 text-white/80" />
                  </span>
                  {f.text}
                </li>
              ))}
            </ul>
          </div>

          <p className="text-xs text-white/40">
            © {new Date().getFullYear()} Aeroxe Enterprises Private limited
          </p>
        </div>
      </div>

      {/* Form panel */}
      <div className="flex flex-1 items-center justify-center bg-surface-50 px-4 py-12 sm:px-8">
        <div className="w-full max-w-sm animate-slide-up">
          <div className="mb-8 lg:hidden">
            <div className="mb-3 flex size-11 items-center justify-center rounded-2xl bg-gradient-to-br from-brand-500 to-brand-700 text-white">
              <Sparkles className="size-5" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-surface-900">LaundryPro</h1>
          </div>

          <h2 className="text-2xl font-bold tracking-tight text-surface-900">Sign in</h2>
          <p className="mt-1.5 text-sm text-surface-500">Enter your credentials to continue.</p>

          <form onSubmit={onSubmit} className="mt-8 space-y-4" noValidate>
            <Field label="Email" required>
              <div className="relative">
                <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-400" />
                <Input
                  type="email"
                  autoComplete="email"
                  placeholder="you@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="pl-9"
                  autoFocus
                />
              </div>
            </Field>

            <Field label="Password" required>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-surface-400" />
                <Input
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pl-9 pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((s) => !s)}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded p-1 text-surface-400 transition-colors hover:text-surface-600"
                >
                  {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
            </Field>

            {error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-xs font-medium text-red-700 animate-fade-in">
                {error}
              </div>
            )}

            <Button type="submit" size="lg" className="w-full" loading={isLoggingIn} disabled={!canSubmit}>
              Sign in
            </Button>
          </form>

          <p className="mt-6 text-center text-xs text-surface-400">
            Protected by role-based access control.
          </p>
        </div>
      </div>
    </div>
  )
}
