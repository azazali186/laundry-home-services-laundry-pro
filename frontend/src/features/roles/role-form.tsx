import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { createRole, updateRole, type CreateRoleRequest, type UpdateRoleRequest } from '@/services/roles.service'
import type { RoleView } from '@/services/auth.service'
import { Button } from '@/components/ui/button'
import { Field, Input, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'
import { PermissionPicker } from './permission-picker'

const schema = z.object({
  name: z.string().min(2, 'Name is required'),
  code: z
    .string()
    .min(2, 'Code is required')
    .regex(/^[a-z0-9_-]+$/, 'Lowercase letters, numbers, - or _'),
})

interface RoleFormProps {
  open: boolean
  role: RoleView | null
  onClose: () => void
}

export function RoleForm({ open, role, onClose }: RoleFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const isEdit = !!role

  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [description, setDescription] = useState('')
  const [permissionIds, setPermissionIds] = useState<string[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setName(role?.name ?? '')
    setCode(role?.code ?? '')
    setDescription(role?.description ?? '')
    setPermissionIds(role?.permissions?.map((p) => p.id) ?? [])
    setErrors({})
  }, [open, role])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && role) {
        const req: UpdateRoleRequest = {
          id: role.id,
          name: name.trim(),
          code: code.trim(),
          description: description.trim() || undefined,
          permission_ids: permissionIds,
        }
        return updateRole(req)
      }
      const req: CreateRoleRequest = {
        name: name.trim(),
        code: code.trim(),
        description: description.trim(),
        permission_ids: permissionIds,
      }
      return createRole(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Role updated' : 'Role created', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['roles'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ name, code })
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
      title={isEdit ? `Edit ${role?.name}` : 'New role'}
      description="Define a role and the permissions it grants."
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending} disabled={role?.is_system && isEdit}>
            {isEdit ? 'Save changes' : 'Create role'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Role name" required error={errors.name}>
            <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </Field>
          <Field label="Code" required error={errors.code} hint="Used in tokens, e.g. branch_manager.">
            <Input value={code} onChange={(e) => setCode(e.target.value)} disabled={isEdit} className="font-mono" />
          </Field>
        </div>
        <Field label="Description">
          <Textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
        </Field>
        <Field label="Permissions" hint="Selected permissions are granted to every user with this role.">
          <PermissionPicker value={permissionIds} onChange={setPermissionIds} />
        </Field>
        {isEdit && role?.is_system && (
          <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs font-medium text-amber-700">
            System roles can be renamed and re-permissioned, but not deleted.
          </p>
        )}
      </form>
    </Modal>
  )
}
