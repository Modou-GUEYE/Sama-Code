import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Statistics, StatusCounts } from '../api/types'
import { useAuth } from '../auth/AuthContext'
import { Alert, Loading, PageHead, useLoad } from '../components/ui'

export function Dashboard() {
  const { me } = useAuth()
  const { data, error, loading } = useLoad(() => api.get<Statistics>('/admin/statistics'), [])

  return (
    <>
      <PageHead title={`Bonjour, ${me?.display_name ?? ''}`} subtitle="Vue d’ensemble des contenus de Sama Code." />
      <Alert>{error}</Alert>
      {loading && !data ? (
        <Loading />
      ) : data ? (
        <div className="stack" style={{ gap: 16 }}>
          <div className="grid cols-3">
            <div className="card stat">
              <b>{data.candidates}</b>
              <span>Candidats inscrits</span>
            </div>
            <div className="card stat">
              <b>{data.exam_sessions}</b>
              <span>Examens blancs passés</span>
            </div>
            <div className="card stat">
              <b>{data.staff}</b>
              <span>Membres de l’équipe</span>
            </div>
          </div>

          <div className="card">
            <h2 style={{ marginBottom: 14 }}>Séries publiées</h2>
            <div className="grid cols-2">
              {data.licenses.map((l) => (
                <div key={l.code} className="stack" style={{ gap: 6 }}>
                  <div className="row between">
                    <b>Permis {l.code}</b>
                    <span className="muted">
                      {l.series_published} / {l.series_expected}
                    </span>
                  </div>
                  <div className="progress">
                    <div style={{ width: `${(100 * l.series_published) / Math.max(l.series_expected, 1)}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="grid cols-2">
            <ContentCard title="Questions" to="/questions" counts={data.questions} />
            <ContentCard title="Séries d’examen" to="/series" counts={data.series} />
            <ContentCard title="Cours" to="/courses" counts={data.courses} />
            <ContentCard title="Leçons" to="/courses" counts={data.lessons} />
          </div>
        </div>
      ) : null}
    </>
  )
}

function ContentCard({ title, to, counts }: { title: string; to: string; counts: StatusCounts }) {
  const total = counts.draft + counts.in_review + counts.published + counts.archived
  return (
    <Link to={to} className="card" style={{ color: 'inherit' }}>
      <div className="row between" style={{ marginBottom: 12 }}>
        <h2>{title}</h2>
        <span className="muted">{total} au total</span>
      </div>
      <div className="row">
        <span className="badge published">{plural(counts.published, 'publié', 'publiés')}</span>
        <span className="badge in_review">{counts.in_review} en validation</span>
        <span className="badge draft">{plural(counts.draft, 'brouillon', 'brouillons')}</span>
      </div>
    </Link>
  )
}

// En français, 0 et 1 sont au singulier.
function plural(n: number, one: string, many: string) {
  return `${n} ${n > 1 ? many : one}`
}
