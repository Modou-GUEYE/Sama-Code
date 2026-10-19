import { useEffect, useState } from 'react'
import { ArrowDown, ArrowLeft, ArrowUp, Check, Plus, X } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { QuestionSummary, SeriesDetail, SeriesQuestion, Status } from '../api/types'
import { StatusActions } from '../components/StatusActions'
import { Alert, Loading, PageHead, StatusBadge, useAction, useLoad } from '../components/ui'

type Item = Pick<SeriesQuestion, 'question_id' | 'text_fr' | 'status' | 'question_type'>

/** Composition d'une série : ordre des questions (glisser par flèches) et publication. */
export function SeriesEditor() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { data: series, error, setData } = useLoad(() => api.get<SeriesDetail>(`/admin/exam-series/${id}`), [id])
  const [items, setItems] = useState<Item[]>([])
  const [dirty, setDirty] = useState(false)
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const bank = useLoad(
    () =>
      api.get<{ items: QuestionSummary[] }>(
        `/admin/questions?status=published&search=${encodeURIComponent(query)}&page_size=50`,
      ),
    [query],
  )
  const action = useAction()

  useEffect(() => {
    if (series) {
      setItems(series.questions)
      setDirty(false)
    }
  }, [series])

  if (error) return <Alert>{error}</Alert>
  if (!series) return <Loading />

  const full = items.length >= series.questions_per_series
  const inSeries = new Set(items.map((q) => q.question_id))

  const update = (next: Item[]) => {
    setItems(next)
    setDirty(true)
  }
  const move = (i: number, delta: number) => {
    const j = i + delta
    if (j < 0 || j >= items.length) return
    const next = [...items]
    ;[next[i], next[j]] = [next[j], next[i]]
    update(next)
  }

  async function saveOrder() {
    const d = await action.run(
      () => api.put<SeriesDetail>(`/admin/exam-series/${id}/questions`, { question_ids: items.map((q) => q.question_id) }),
      'Composition enregistrée.',
    )
    if (d) setData(d)
  }

  async function changeStatus(to: Status) {
    const d = await action.run(
      () => api.post<SeriesDetail>(`/admin/exam-series/${id}/status`, { status: to }),
      to === 'published' ? 'Série publiée : une nouvelle version est figée pour les candidats.' : 'Statut mis à jour.',
    )
    if (d) setData(d)
  }

  return (
    <>
      <PageHead
        title={`Permis ${series.license_code} — Série ${series.series_number}`}
        subtitle={`${items.length} / ${series.questions_per_series} questions${
          series.current_version > 0 ? ` · version publiée : v${series.current_version}` : ''
        }`}
        actions={
          <>
            <StatusBadge status={series.status} />
            <button className="btn ghost" onClick={() => navigate('/series')}>
              <ArrowLeft size={16} /> Retour
            </button>
          </>
        }
      />
      <Alert>{action.error}</Alert>
      <Alert kind="success">{action.success}</Alert>
      {series.status === 'published' && (
        <Alert kind="info">
          Série publiée. Modifier sa composition la repasse en brouillon ; les candidats gardent la version v
          {series.current_version} jusqu’à la prochaine publication.
        </Alert>
      )}

      <div className="grid cols-2" style={{ marginTop: 16, alignItems: 'start' }}>
        <div className="card stack">
          <div className="row between">
            <h2>Ordre des questions</h2>
            <div className="row">
              <button className="btn" disabled={!dirty || action.busy} onClick={() => void saveOrder()}>
                Enregistrer l’ordre
              </button>
              {!dirty && <StatusActions status={series.status} busy={action.busy} onChange={(to) => void changeStatus(to)} />}
            </div>
          </div>
          {items.length === 0 && <div className="empty">Ajoutez des questions publiées depuis la banque.</div>}
          {items.map((q, i) => (
            <div key={q.question_id} className="order-item">
              <span className="pos">{i + 1}</span>
              <Link to={`/questions/${q.question_id}`} className="ellipsis" style={{ color: 'inherit', maxWidth: 360 }}>
                {q.text_fr}
              </Link>
              <div className="row" style={{ gap: 2 }}>
                {q.status !== 'published' && <StatusBadge status={q.status} />}
                <button className="btn ghost small icon-only" title="Monter" disabled={i === 0} onClick={() => move(i, -1)}>
                  <ArrowUp size={16} />
                </button>
                <button
                  className="btn ghost small icon-only"
                  title="Descendre"
                  disabled={i === items.length - 1}
                  onClick={() => move(i, 1)}
                >
                  <ArrowDown size={16} />
                </button>
                <button className="btn ghost small icon-only" title="Retirer" onClick={() => update(items.filter((_, j) => j !== i))}>
                  <X size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className="card stack">
          <h2>Banque de questions publiées</h2>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              setQuery(search)
            }}
          >
            <input className="input" placeholder="Rechercher…" value={search} onChange={(e) => setSearch(e.target.value)} />
          </form>
          {full && <Alert kind="info">La série est complète ({series.questions_per_series} questions).</Alert>}
          {bank.data?.items.length === 0 && <div className="empty">Aucune question publiée trouvée.</div>}
          {bank.data?.items.map((q) => (
            <div key={q.id} className="order-item" style={{ gridTemplateColumns: '1fr auto' }}>
              <span className="ellipsis" style={{ maxWidth: 420 }}>
                {q.text_fr}
              </span>
              <button
                className="btn secondary small"
                disabled={full || inSeries.has(q.id)}
                onClick={() =>
                  update([...items, { question_id: q.id, text_fr: q.text_fr, status: q.status, question_type: q.question_type }])
                }
              >
                {inSeries.has(q.id) ? (
                  <>
                    <Check size={15} /> Ajoutée
                  </>
                ) : (
                  <>
                    <Plus size={15} strokeWidth={2.4} /> Ajouter
                  </>
                )}
              </button>
            </div>
          ))}
        </div>
      </div>
    </>
  )
}
