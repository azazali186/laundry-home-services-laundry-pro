import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, Search } from 'lucide-react'
import { fetchAllPages } from '@/lib/fetchAll'
import { cn, humanize } from '@/lib/utils'
import { listPermissions } from '@/services/roles.service'
import type { PermissionView } from '@/services/auth.service'
import { Input } from '@/components/ui/form'

interface PermissionPickerProps {
  value: string[]
  onChange: (ids: string[]) => void
}

/** Grouped permission checklist grouped by service/entity. */
export function PermissionPicker({ value, onChange }: PermissionPickerProps) {
  const [query, setQuery] = useState('')
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

  const queryPerms = useQuery({
    queryKey: ['permissions', 'all'],
    queryFn: () => fetchAllPages<PermissionView>(listPermissions),
    staleTime: 30 * 60_000,
  })

  const grouped = useMemo(() => {
    const perms = queryPerms.data ?? []
    const filtered = query
      ? perms.filter(
          (p) =>
            p.name.toLowerCase().includes(query.toLowerCase()) ||
            p.route.toLowerCase().includes(query.toLowerCase()),
        )
      : perms
    const groups = new Map<string, PermissionView[]>()
    for (const p of filtered) {
      const key = p.service || 'misc'
      groups.set(key, [...(groups.get(key) ?? []), p])
    }
    return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  }, [queryPerms.data, query])

  function toggle(id: string) {
    onChange(value.includes(id) ? value.filter((v) => v !== id) : [...value, id])
  }

  function toggleGroup(perms: PermissionView[]) {
    const ids = perms.map((p) => p.id)
    const allSelected = ids.every((id) => value.includes(id))
    onChange(allSelected ? value.filter((v) => !ids.includes(v)) : [...new Set([...value, ...ids])])
  }

  if (queryPerms.isLoading) {
    return <p className="py-6 text-center text-xs text-surface-400">Loading permissions…</p>
  }

  if (queryPerms.isError) {
    return <p className="py-6 text-center text-xs text-red-500">Failed to load permissions.</p>
  }

  return (
    <div className="space-y-2">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-surface-400" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Filter permissions…"
          className="h-8.5 pl-8 text-xs"
        />
      </div>
      <div className="max-h-72 space-y-1.5 overflow-y-auto rounded-lg border border-surface-200 p-2">
        {grouped.length === 0 && (
          <p className="py-6 text-center text-xs text-surface-400">No permissions match.</p>
        )}
        {grouped.map(([group, perms]) => {
          const ids = perms.map((p) => p.id)
          const allSelected = ids.length > 0 && ids.every((id) => value.includes(id))
          const isCollapsed = collapsed.has(group)
          return (
            <div key={group}>
              <button
                type="button"
                onClick={() => {
                  setCollapsed((prev) => {
                    const next = new Set(prev)
                    if (next.has(group)) next.delete(group)
                    else next.add(group)
                    return next
                  })
                }}
                className="flex w-full items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-xs font-semibold text-surface-700 hover:bg-surface-100"
              >
                {isCollapsed ? <ChevronRight className="size-3.5 text-surface-400" /> : <ChevronDown className="size-3.5 text-surface-400" />}
                {humanize(group)}
                <span className="ml-auto flex items-center gap-2">
                  <span className="text-[10px] font-normal text-surface-400">
                    {value.filter((id) => ids.includes(id)).length}/{ids.length}
                  </span>
                  <span
                    role="checkbox"
                    aria-checked={allSelected}
                    onClick={(e) => {
                      e.stopPropagation()
                      toggleGroup(perms)
                    }}
                    className={cn(
                      'rounded border px-1.5 py-0.5 text-[10px] font-semibold transition-colors',
                      allSelected
                        ? 'border-brand-600 bg-brand-600 text-white'
                        : 'border-surface-300 text-surface-500 hover:border-brand-400',
                    )}
                  >
                    {allSelected ? 'All' : 'Select'}
                  </span>
                </span>
              </button>
              {!isCollapsed && (
                <div className="ml-3 mt-0.5 space-y-0.5 border-l border-surface-100 pl-3">
                  {perms.map((p) => (
                    <label
                      key={p.id}
                      className="flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-xs text-surface-600 hover:bg-surface-100"
                    >
                      <input
                        type="checkbox"
                        checked={value.includes(p.id)}
                        onChange={() => toggle(p.id)}
                        className="size-3.5 accent-brand-600"
                      />
                      <span className="font-mono text-[10px] text-surface-400">{p.method}</span>
                      <span className="truncate">{p.name}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
