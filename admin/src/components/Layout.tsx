import { NavLink, Outlet } from 'react-router-dom'
import { BookOpen, Car, CircleHelp, ClipboardList, LayoutDashboard, LogOut, Users } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'

export function Layout() {
  const { logout, can } = useAuth()

  const links = [
    { to: '/', icon: LayoutDashboard, label: 'Tableau de bord', end: true },
    { to: '/questions', icon: CircleHelp, label: 'Questions' },
    { to: '/series', icon: ClipboardList, label: 'Séries d’examen' },
    { to: '/courses', icon: BookOpen, label: 'Cours' },
    ...(can('admin')
      ? [
          { to: '/users', icon: Users, label: 'Utilisateurs' },
          { to: '/licenses', icon: Car, label: 'Permis' },
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
              <l.icon className="icon" size={20} strokeWidth={2.2} />
              {l.label}
            </NavLink>
          ))}
        </nav>
        <div className="spacer" />
        <button type="button" className="nav-logout" onClick={() => void logout()}>
          <LogOut size={20} strokeWidth={2.2} />
          Se déconnecter
        </button>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
