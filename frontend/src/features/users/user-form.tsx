import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { cn } from '@/lib/utils'
import { createUser, updateUser, type CreateUserRequest, type UpdateUserRequest } from '@/services/users.service'
import type { UserView } from '@/services/auth.service'
import { useRoleOptions } from '@/hooks/useRoleOptions'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const createSchema = z.object({
  name: z.string().min(2, 'Name is required'),
  email: z.string().email('Enter a valid email'),
  phone: z.string().optional(),
  password: z.string().min(8, 'Minimum 8 characters'),
})

const editSchema = z.object({
  name: z.string().min(2, 'Name is required'),
  phone: z.string().optional(),
  password: z
    .union([z.literal(''), z.string().min(8, 'Minimum 8 characters')])
    .optional(),
})

interface UserFormProps {
  open: boolean
  /** When set, the form edits this user; otherwise it creates. */
  user: UserView | null
  onClose: () => void
}

export function UserForm({ open, user, onClose }: UserFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const roles = useRoleOptions()
  const isEdit = !!user

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setName(user?.name ?? '')
    setEmail(user?.email ?? '')
    setPhone(user?.phone ?? '')
    setPassword('')
    setIsActive(user?.is_active ?? true)
    setSelectedRoles(user?.roles?.map((r) => r.id) ?? [])
    setErrors({})
  }, [open, user])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && user) {
        const req: UpdateUserRequest = {
          id: user.id,
          name: name.trim(),
          phone: phone.trim() || undefined,
          password: password || undefined,
          role_ids: selectedRoles,
        }
        return updateUser(req)
      }
      const req: CreateUserRequest = {
        name: name.trim(),
        email: email.trim(),
        phone: phone.trim(),
        password,
        role_ids: selectedRoles,
      }
      return createUser(req)
    },
    onSuccess: () => {
      toast.toast({
        title: isEdit ? 'User updated' : 'User created',
        variant: 'success',
      })
      void queryClient.invalidateQueries({ queryKey: ['users'] })
      onClose()
    },
    onError: (err) => {
      toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' })
    },
  })

  function toggleRole(id: string) {
    setSelectedRoles((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id],
    )
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = (isEdit ? editSchema : createSchema).safeParse({
      name,
      email,
      phone,
      password,
    })
    if (!result.success) {
      const fieldErrors: Record<string, string> = {}
      for (const issue of result.error.issues) fieldErrors[issue.path[0] as string] = issue.message
      setErrors(fieldErrors)
      return
    }
    setErrors({})
    void mutation.mutate()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? `Edit ${user?.name}` : 'New user'}
      description={isEdit ? 'Update profile, status and role assignments.' : 'Invite a team member to the console.'}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Create user'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <Field label="Full name" required error={errors.name}>
          <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        </Field>
        <Field label="Email" required error={errors.email}>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} disabled={isEdit} />
        </Field>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Phone" error={errors.phone}>
            <Input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+1 555 000 0000" />
          </Field>
          <Field label={isEdit ? 'New password' : 'Password'} required={!isEdit} error={errors.password} hint={isEdit ? 'Leave blank to keep current.' : 'Minimum 8 characters.'}>
            <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
          </Field>
        </div>

        <Field label="Roles" hint="Users inherit permissions from their roles.">
          <div className="grid max-h-44 grid-cols-1 gap-1 overflow-y-auto rounded-lg border border-surface-200 p-2 sm:grid-cols-2">
            {roles.isLoading && <p className="col-span-full px-2 py-1.5 text-xs text-surface-400">Loading roles…</p>}
            {roles.data?.map((role) => (
              <label
                key={role.id}
                className={cn(
                  'flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-2 text-xs font-medium transition-colors',
                  selectedRoles.includes(role.id)
                    ? 'bg-brand-50 text-brand-700'
                    : 'text-surface-600 hover:bg-surface-100',
                )}
              >
                <input
                  type="checkbox"
                  checked={selectedRoles.includes(role.id)}
                  onChange={() => toggleRole(role.id)}
                  className="size-3.5 accent-brand-600"
                />
                {role.name}
                {role.is_system && <span className="ml-auto text-[10px] text-surface-400">system</span>}
              </label>
            ))}
            {!roles.isLoading && roles.data?.length === 0 && (
              <p className="col-span-full px-2 py-1.5 text-xs text-surface-400">No roles available.</p>
            )}
          </div>
        </Field>

        {isEdit && (
          <label className="flex cursor-pointer items-center gap-2.5 text-sm text-surface-700">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="size-4 accent-brand-600"
            />
            <span className="text-xs font-medium">Account active</span>
          </label>
        )}
      </form>
    </Modal>
  )
}
