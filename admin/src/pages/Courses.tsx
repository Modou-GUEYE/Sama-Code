import { useState } from 'react'
import { Pencil, Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { Course, CourseCategory, Translations } from '../api/types'
import { DeleteButton } from '../components/DeleteButton'
import { cleanTranslations, TranslationFields } from '../components/TranslationFields'
import { Alert, Field, Loading, PageHead, StatusBadge, useAction, useLoad } from '../components/ui'

/** Thèmes (panneaux, priorités…) et leurs cours. */
export function Courses() {
  const navigate = useNavigate()
  const categories = useLoad(() => api.get<{ items: CourseCategory[] }>('/admin/course-categories'), [])
  const [selected, setSelected] = useState<string>('')
  const category = categories.data?.items.find((c) => c.id === selected) ?? categories.data?.items[0]
  const courses = useLoad(
    () => (category ? api.get<{ items: Course[] }>(`/admin/courses?category_id=${category.id}`) : Promise.resolve({ items: [] })),
    [category?.id],
  )
  const [editing, setEditing] = useState<CourseCategory | 'new' | null>(null)
  const action = useAction()

  async function createCourse() {
    if (!category) return
    const c = await action.run(() =>
      api.post<Course>('/admin/courses', {
        category_id: category.id,
        sort_order: (courses.data?.items.length ?? 0) + 1,
        translations: { fr: { title: 'Nouveau cours', audio_media_id: null } },
      }),
    )
    if (c) navigate(`/courses/${c.id}`)
  }

  if (categories.loading && !categories.data) return <Loading />

  return (
    <>
      <PageHead
        title="Cours"
        subtitle="Thèmes → cours → leçons, à lire ou à écouter en français et en wolof."
        actions={
          <button className="btn secondary" onClick={() => setEditing('new')}>
            <Plus size={16} strokeWidth={2.4} /> Nouveau thème
          </button>
        }
      />
      <Alert>{categories.error ?? courses.error ?? action.error}</Alert>

      {editing && (
        <CategoryForm
          category={editing === 'new' ? null : editing}
          count={categories.data?.items.length ?? 0}
          onDone={(id) => {
            setEditing(null)
            if (id) setSelected(id)
            void categories.reload()
          }}
        />
      )}

      {categories.data?.items.length === 0 && !editing ? (
        <div className="card empty">Commencez par créer un thème (ex. « Panneaux de signalisation »).</div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: '280px 1fr', gap: 16, alignItems: 'start' }}>
          <div className="card stack" style={{ padding: 10, gap: 4 }}>
            {categories.data?.items.map((c) => (
              <button
                key={c.id}
                type="button"
                className="btn ghost"
                style={{
                  justifyContent: 'space-between',
                  borderRadius: 14,
                  height: 'auto',
                  padding: '10px 14px',
                  background: c.id === category?.id ? 'var(--green-pale)' : undefined,
                  color: c.id === category?.id ? 'var(--green-dark)' : 'var(--ink)',
                }}
                onClick={() => setSelected(c.id)}
              >
                <span style={{ whiteSpace: 'normal', textAlign: 'left' }}>{c.translations.fr?.title ?? c.code}</span>
                <span className="muted small">{c.course_count}</span>
              </button>
            ))}
          </div>

          {category && (
            <div className="card stack">
              <div className="row between">
                <div>
                  <h2>{category.translations.fr?.title}</h2>
                  <span className="muted small">
                    {category.translations.wo?.title ?? <span className="badge lang">Wolof manquant</span>}
                  </span>
                </div>
                <div className="row">
                  <button className="btn ghost small" onClick={() => setEditing(category)}>
                    <Pencil size={15} /> Modifier le thème
                  </button>
                  <button className="btn small" disabled={action.busy} onClick={() => void createCourse()}>
                    <Plus size={16} strokeWidth={2.4} /> Nouveau cours
                  </button>
                </div>
              </div>
              {courses.data?.items.length === 0 && <div className="empty">Aucun cours dans ce thème.</div>}
              <table className="table">
                <tbody>
                  {courses.data?.items.map((c) => (
                    <tr key={c.id} className="clickable" onClick={() => navigate(`/courses/${c.id}`)}>
                      <td>
                        <b>{c.translations.fr?.title}</b>
                        <div className="muted small">{c.translations.wo?.title ?? 'Wolof manquant'}</div>
                      </td>
                      <td className="muted">{c.lesson_count} leçon(s)</td>
                      <td style={{ textAlign: 'right' }}>
                        <StatusBadge status={c.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </>
  )
}

function CategoryForm({
  category,
  count,
  onDone,
}: {
  category: CourseCategory | null
  count: number
  onDone: (id?: string) => void
}) {
  const [code, setCode] = useState(category?.code ?? '')
  const [translations, setTranslations] = useState<Translations>(category?.translations ?? {})
  const action = useAction()

  async function remove() {
    if (!category) return
    const ok = await action.run(() => api.del(`/admin/course-categories/${category.id}`).then(() => true))
    if (ok) onDone()
  }

  async function save() {
    const body = {
      code,
      icon_media_id: category?.icon_media_id ?? null,
      sort_order: category?.sort_order ?? count + 1,
      translations: cleanTranslations(translations),
    }
    if (category) {
      const ok = await action.run(() => api.put(`/admin/course-categories/${category.id}`, body))
      if (ok !== undefined || !action.error) onDone(category.id)
    } else {
      const created = await action.run(() => api.post<{ id: string }>('/admin/course-categories', body))
      if (created) onDone(created.id)
    }
  }

  return (
    <div className="card stack" style={{ marginBottom: 16 }}>
      <h2>{category ? 'Modifier le thème' : 'Nouveau thème'}</h2>
      <Field label="Code (identifiant technique, ex. panneaux)">
        <input className="input" value={code} onChange={(e) => setCode(e.target.value.toLowerCase())} />
      </Field>
      <TranslationFields value={translations} onChange={setTranslations} titleLabel="Nom" withDescription />
      <Alert>{action.error}</Alert>
      <div className="row">
        <button className="btn" disabled={action.busy} onClick={() => void save()}>
          Enregistrer
        </button>
        <button className="btn ghost" onClick={() => onDone()}>
          Annuler
        </button>
        {category && (
          <>
            <span style={{ flex: 1 }} />
            <DeleteButton
              what={`le thème « ${category.translations.fr?.title ?? category.code} »`}
              busy={action.busy}
              onConfirm={() => void remove()}
            />
          </>
        )}
      </div>
    </div>
  )
}
