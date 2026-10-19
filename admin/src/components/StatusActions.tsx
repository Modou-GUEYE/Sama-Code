import { useAuth } from '../auth/AuthContext'
import type { Status } from '../api/types'

interface Transition {
  to: Status
  label: string
  kind: 'primary' | 'secondary' | 'danger'
  roles: ('editor' | 'validator')[]
}

// Même règle que l'API (internal/workflow) : l'admin peut tout.
const TRANSITIONS: Record<Status, Transition[]> = {
  draft: [{ to: 'in_review', label: 'Soumettre à validation', kind: 'primary', roles: ['editor', 'validator'] }],
  in_review: [
    { to: 'published', label: 'Publier', kind: 'primary', roles: ['validator'] },
    { to: 'draft', label: 'Renvoyer en brouillon', kind: 'secondary', roles: ['editor', 'validator'] },
  ],
  published: [
    { to: 'draft', label: 'Dépublier', kind: 'secondary', roles: ['validator'] },
    { to: 'archived', label: 'Archiver', kind: 'danger', roles: ['validator'] },
  ],
  archived: [{ to: 'draft', label: 'Restaurer en brouillon', kind: 'secondary', roles: [] }],
}

export function StatusActions({
  status,
  busy,
  onChange,
}: {
  status: Status
  busy?: boolean
  onChange: (to: Status) => void
}) {
  const { can } = useAuth()
  const actions = TRANSITIONS[status].filter((t) => can(...t.roles))
  if (actions.length === 0) return null
  return (
    <>
      {actions.map((t) => (
        <button
          key={t.to}
          type="button"
          className={`btn ${t.kind === 'primary' ? '' : t.kind}`}
          disabled={busy}
          onClick={() => {
            if (t.kind === 'danger' && !confirm(`${t.label} ?`)) return
            onChange(t.to)
          }}
        >
          {t.label}
        </button>
      ))}
    </>
  )
}
