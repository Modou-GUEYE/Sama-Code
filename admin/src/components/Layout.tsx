import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

const ROLE_LABEL: Record<string, string> = {
  editor: 'Éditeur',
  validator: 'Validateur',
  admin: 'Administrateur',
}

export function Layout() {
  const { me, logout, can } = useAuth()
  const roles = me?.roles.filter((r) => r !== 'candidate').map((r) => ROLE_LABEL[r]).join(' · ')

  const links = [
    { to: '/', icon: '📊', label: 'Tableau de bord', end: true },
    { to: '/questions', icon: '❓', label: 'Questions' },
    { to: '/series', icon: '📝', label: 'Séries d’examen' },
    { to: '/courses', icon: '📚', label: 'Cours' },
    ...(can('admin')
      ? [
          { to: '/users', icon: '👥', label: 'Utilisateurs' },
          { to: '/licenses', icon: '🚗', label: 'Permis' },
        ]
      : []),
  ]

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <img src="/logo.png" alt="" />
          <div>
            <b>Sama Code</b>
            <span>Administration</span>
          </div>
        </div>
        <nav className="nav">
          {links.map((l) => (
            <NavLink key={l.to} to={l.to} end={l.end}>
              <span className="icon">{l.icon}</span>
              {l.label}
            </NavLink>
          ))}
        </nav>
        <div className="spacer" />
        <div className="me">
          <b>{me?.display_name}</b>
          <small>{roles}</small>
          <button type="button" className="btn ghost small" style={{ paddingLeft: 0 }} onClick={() => void logout()}>
            Se déconnecter
          </button>
        </div>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
