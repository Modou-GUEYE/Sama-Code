import { useState } from 'react'
import { ChevronLeft, ChevronRight, Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { QuestionSummary, Status } from '../api/types'
import { Alert, Loading, PageHead, StatusBadge, Tabs, useLoad } from '../components/ui'

type Filter = Status | ''
const PAGE_SIZE = 20

export function Questions() {
  const navigate = useNavigate()
  const [status, setStatus] = useState<Filter>('')
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const [page, setPage] = useState(1)

  const { data, error, loading } = useLoad(
    () =>
      api.get<{ items: QuestionSummary[]; total: number }>(
        `/admin/questions?status=${status}&search=${encodeURIComponent(query)}&page=${page}&page_size=${PAGE_SIZE}`,
      ),
    [status, query, page],
  )
  const pages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  return (
    <>
      <PageHead
        title="Questions"
        subtitle="Banque de questions utilisée par les séries d’examen et l’entraînement."
        actions={
          <button className="btn" onClick={() => navigate('/questions/new')}>
            <Plus size={18} strokeWidth={2.4} /> Nouvelle question
          </button>
        }
      />
      <div className="row between" style={{ marginBottom: 16 }}>
        <Tabs<Filter>
          value={status}
          onChange={(v) => {
            setStatus(v)
            setPage(1)
          }}
          options={[
            { value: '', label: 'Toutes' },
            { value: 'draft', label: 'Brouillons' },
            { value: 'in_review', label: 'En validation' },
            { value: 'published', label: 'Publiées' },
            { value: 'archived', label: 'Archivées' },
          ]}
        />
        <form
          onSubmit={(e) => {
            e.preventDefault()
            setQuery(search)
            setPage(1)
          }}
          style={{ width: 320 }}
        >
          <input
            className="input"
            placeholder="Rechercher dans l’énoncé…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ background: '#fff' }}
          />
        </form>
      </div>
      <Alert>{error}</Alert>
      <div className="card" style={{ padding: 8 }}>
        {loading && !data ? (
          <Loading />
        ) : data && data.items.length > 0 ? (
          <table className="table">
            <thead>
              <tr>
                <th>Énoncé</th>
                <th>Type</th>
                <th>Langues</th>
                <th>Séries</th>
                <th>Statut</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((q) => (
                <tr key={q.id} className="clickable" onClick={() => navigate(`/questions/${q.id}`)}>
                  <td className="ellipsis">{q.text_fr}</td>
                  <td className="muted">{q.question_type === 'single' ? 'Unique' : 'Multiple'}</td>
                  <td>
                    <span className="badge published">FR</span>{' '}
                    {q.has_wolof ? <span className="badge published">Audio wolof</span> : <span className="badge lang">Audio wolof manquant</span>}
                  </td>
                  <td className="muted">{q.series_count}</td>
                  <td>
                    <StatusBadge status={q.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="empty">Aucune question pour l’instant.</div>
        )}
      </div>
      {pages > 1 && (
        <div className="row" style={{ justifyContent: 'center', marginTop: 16 }}>
          <button className="btn secondary small" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            <ChevronLeft size={16} /> Précédent
          </button>
          <span className="muted">
            Page {page} / {pages}
          </span>
          <button className="btn secondary small" disabled={page >= pages} onClick={() => setPage(page + 1)}>
            Suivant <ChevronRight size={16} />
          </button>
        </div>
      )}
    </>
  )
}
