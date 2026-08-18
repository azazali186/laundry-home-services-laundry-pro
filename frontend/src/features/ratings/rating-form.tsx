import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorMessage } from '@/lib/api'
import { fetchAllPages } from '@/lib/fetchAll'
import {
  createRating,
  updateRating,
  type CreateRatingRequest,
  type UpdateRatingRequest,
  type RatingView,
} from '@/services/ratings.service'
import { listUsers } from '@/services/users.service'
import type { UserView } from '@/services/auth.service'
import { useOrderOptions } from '@/hooks/useOrderOptions'
import { Button } from '@/components/ui/button'
import { Field, Select, Textarea } from '@/components/ui/form'
import { Modal } from '@/components/ui/modal'
import { useToast } from '@/components/ui/toast'

const schema = z.object({
  order_id: z.string().min(1, 'Order is required'),
  customer_id: z.string().min(1, 'Customer is required'),
  score: z.coerce.number().int().min(1).max(5),
})

const SCORES = [
  { value: '5', label: '5 — Excellent' },
  { value: '4', label: '4 — Good' },
  { value: '3', label: '3 — Average' },
  { value: '2', label: '2 — Poor' },
  { value: '1', label: '1 — Terrible' },
]

interface RatingFormProps {
  open: boolean
  rating: RatingView | null
  onClose: () => void
}

export function RatingForm({ open, rating, onClose }: RatingFormProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const orders = useOrderOptions()
  const isEdit = !!rating

  const customersQuery = useQuery({
    queryKey: ['users', 'all'],
    queryFn: () => fetchAllPages<UserView>(listUsers, { limit: 100, sort_by: 'name' }),
    staleTime: 5 * 60_000,
  })

  const [orderId, setOrderId] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [score, setScore] = useState('5')
  const [comment, setComment] = useState('')
  const [isPublic, setIsPublic] = useState('true')
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setOrderId(rating?.order_id ?? '')
    setCustomerId(rating?.customer_id ?? '')
    setScore(rating ? String(rating.score) : '5')
    setComment(rating?.comment ?? '')
    setIsPublic(rating ? String(rating.is_public) : 'true')
    setErrors({})
  }, [open, rating])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && rating) {
        const req: UpdateRatingRequest = {
          id: rating.id,
          score: Number(score),
          comment: comment.trim() || undefined,
          is_public: isPublic === 'true',
        }
        return updateRating(req)
      }
      const req: CreateRatingRequest = {
        order_id: orderId,
        customer_id: customerId,
        score: Number(score),
        comment: comment.trim() || undefined,
        is_public: isPublic === 'true',
      }
      return createRating(req)
    },
    onSuccess: () => {
      toast.toast({ title: isEdit ? 'Rating updated' : 'Rating recorded', variant: 'success' })
      void queryClient.invalidateQueries({ queryKey: ['ratings'] })
      onClose()
    },
    onError: (err) => toast.toast({ title: 'Save failed', description: errorMessage(err), variant: 'error' }),
  })

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const result = schema.safeParse({ order_id: orderId, customer_id: customerId, score })
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
      title={isEdit ? 'Edit rating' : 'Record rating'}
      description={isEdit ? 'Update the score and comment.' : 'Log customer feedback for an order.'}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={onSubmit} loading={mutation.isPending}>
            {isEdit ? 'Save changes' : 'Record rating'}
          </Button>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Order" required error={errors.order_id}>
            <Select value={orderId} onChange={(e) => setOrderId(e.target.value)} disabled={isEdit}>
              <option value="">Select order…</option>
              {orders.data?.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.order_no}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Customer" required error={errors.customer_id}>
            <Select value={customerId} onChange={(e) => setCustomerId(e.target.value)} disabled={isEdit}>
              <option value="">Select customer…</option>
              {customersQuery.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name} — {u.email}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Score" required error={errors.score}>
            <Select value={score} onChange={(e) => setScore(e.target.value)}>
              {SCORES.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Visibility">
            <Select value={isPublic} onChange={(e) => setIsPublic(e.target.value)}>
              <option value="true">Public</option>
              <option value="false">Private</option>
            </Select>
          </Field>
        </div>

        <Field label="Comment">
          <Textarea value={comment} onChange={(e) => setComment(e.target.value)} rows={3} placeholder="What did the customer say?" />
        </Field>
      </form>
    </Modal>
  )
}
