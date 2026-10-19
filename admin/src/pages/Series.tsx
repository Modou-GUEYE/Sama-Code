import { useState } from 'react'
import { Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { License, Series as SeriesT, SeriesDetail } from '../api/types'
import { Alert, Loading, PageHead, StatusBadge, Tabs, useAction, useLoad } from '../components/ui'

/** Grille des séries de chaque permis : une case par série attendue (CDC §5.1). */
export function Series() {
  const navigate = useNavigate()
  const licenses = useLoad(() => api.get<{ items: License[] }>('/admin/license-categories'), [])
  const [licenseId, setLicenseId] = useState<string>('')
  const license = licenses.data?.items.find((l) => l.id === licenseId) ?? licenses.data?.items[0]
  const series = useLoad(
    () => (license ? api.get<{ items: SeriesT[] }>(`/admin/exam-series?license_category_id=${license.id}`) : Promise.resolve({ items: [] })),
    [license?.id],
  )
  const create = useAction()

  async function open(number: number) {
    const existing = series.data?.items.find((s) => s.series_number === number)
    if (existing) {
      navigate(`/series/${existing.id}`)
      return
    }
    if (!license) return
    const created = await create.run(() =>
      api.post<SeriesDetail>('/admin/exam-series', { license_category_id: license.id, series_number: number }),
    )
    if (created) navigate(`/series/${created.id}`)
  }

  if (licenses.loading && !licenses.data) return <Loading />

  return (
    <>
      <PageHead
        title="Séries d’examen"
        subtitle="Séries fixes : le candidat choisit sa série, l’ordre des questions ne change jamais."
        actions={
          licenses.data && (
            <Tabs
              value={license?.id ?? ''}
              onChange={setLicenseId}
              options={licenses.data.items.map((l) => ({ value: l.id, label: `Permis ${l.code}` }))}
            />
          )
        }
      />
      <Alert>{licenses.error ?? series.error ?? create.error}</Alert>
      {license && (
        <p className="muted" style={{ marginTop: 0 }}>
          {license.series_count} séries de {license.questions_per_series} questions · {license.max_faults} fautes
          autorisées · {license.exam_duration_minutes} minutes
        </p>
      )}
      <div className="series-grid">
        {license &&
          Array.from({ length: license.series_count }, (_, i) => i + 1).map((n) => {
            const s = series.data?.items.find((x) => x.series_number === n)
            return (
              <button
                key={n}
                type="button"
                className={`card series-tile ${s ? '' : 'missing'}`}
                disabled={create.busy}
                onClick={() => void open(n)}
              >
                <div className="row between">
                  <span className="num">{n}</span>
                  {s && <StatusBadge status={s.status} />}
                </div>
                {s ? (
                  <>
                    <div className="small muted" style={{ margin: '8px 0 6px' }}>
                      {s.question_count} / {s.questions_per_series} questions
                      {s.current_version > 0 && ` · v${s.current_version}`}
                    </div>
                    <div className="progress">
                      <div style={{ width: `${(100 * s.question_count) / s.questions_per_series}%` }} />
                    </div>
                  </>
                ) : (
                  <div className="small" style={{ marginTop: 8 }}>
                    <Plus size={14} strokeWidth={2.4} style={{ verticalAlign: '-2px' }} /> Créer la série
                  </div>
                )}
              </button>
            )
          })}
      </div>
    </>
  )
}
