import { useState } from 'react'
import { api } from '../api/client'
import type { AdminUser, Role } from '../api/types'
import { useAuth } from '../auth/AuthContext'
import { Alert, Loading, PageHead, useAction, useLoad } from '../components/ui'

const ROLES: { value: Role; label: string }[] = [
  { value: 'candidate', label: 'Candidat' },
  { value: 'editor', label: 'Éditeur' },
  { value: 'validator', label: 'Validateur' },
  { value: 'admin', label: 'Admin' },
]

/** Comptes et rôles (administrateur uniquement). */
export function Users() {
  const { me } = useAuth()
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const { data, error, loading, setData } = useLoad(
    () => api.get<{ items: AdminUser[] }>(`/admin/users?search=${encodeURIComponent(query)}&page_size=50`),
    [query],
  )
  const action = useAction()

  async function toggle(user: AdminUser, role: Role) {
    const roles = user.roles.includes(role) ? user.roles.filter((r) => r !== role) : [...user.roles, role]
    const out = await action.run(
      () => api.put<{ roles: Role[] }>(`/admin/users/${user.id}/roles`, { roles }),
      `Rôles de ${user.display_name} mis à jour (effectif sous 15 minutes).`,
    )
    if (out && data) setData({ items: data.items.map((u) => (u.id === user.id ? { ...u, roles: out.roles } : u)) })
  }

  return (
    <>
      <PageHead title="Utilisateurs" subtitle="Attribuez les rôles d’éditeur, de validateur et d’administrateur." />
      <form
        onSubmit={(e) => {
          e.preventDefault()
          setQuery(search)
        }}
        style={{ maxWidth: 360, marginBottom: 16 }}
      >
        <input
          className="input"
          style={{ background: '#fff' }}
          placeholder="Nom ou numéro…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </form>
      <Alert>{error ?? action.error}</Alert>
      <Alert kind="success">{action.success}</Alert>
      <div className="card" style={{ padding: 8, marginTop: 12 }}>
        {loading && !data ? (
          <Loading />
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Nom</th>
                <th>Téléphone</th>
                {ROLES.map((r) => (
                  <th key={r.value} style={{ textAlign: 'center' }}>
                    {r.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data?.items.map((u) => (
                <tr key={u.id}>
                  <td>
                    <b>{u.display_name}</b>
                    {u.id === me?.id && <span className="muted small"> (vous)</span>}
                  </td>
                  <td className="muted">{u.phone_number}</td>
                  {ROLES.map((r) => (
                    <td key={r.value} style={{ textAlign: 'center' }}>
                      <input
                        type="checkbox"
                        style={{ width: 18, height: 18, accentColor: 'var(--green)' }}
                        checked={u.roles.includes(r.value)}
                        disabled={action.busy || (u.id === me?.id && r.value === 'admin')}
                        onChange={() => void toggle(u, r.value)}
                      />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  )
}
