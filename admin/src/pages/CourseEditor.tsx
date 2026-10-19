import { useEffect, useState } from 'react'
import { ArrowLeft, Plus } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Course, CourseCategory, Lesson, License, Status, Translations } from '../api/types'
import { DeleteButton } from '../components/DeleteButton'
import { StatusActions } from '../components/StatusActions'
import { cleanTranslations, TranslationFields } from '../components/TranslationFields'
import { Alert, Field, Loading, PageHead, StatusBadge, useAction, useLoad } from '../components/ui'

export function CourseEditor() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { data: course, error, setData } = useLoad(() => api.get<Course>(`/admin/courses/${id}`), [id])
  const categories = useLoad(() => api.get<{ items: CourseCategory[] }>('/admin/course-categories'), [])
  const licenses = useLoad(() => api.get<{ items: License[] }>('/admin/license-categories'), [])
  const [translations, setTranslations] = useState<Translations>({})
  const [categoryId, setCategoryId] = useState('')
  const [licenseId, setLicenseId] = useState<string>('')
  const action = useAction()

  useEffect(() => {
    if (!course) return
    setTranslations(course.translations)
    setCategoryId(course.category_id)
    setLicenseId(course.license_category_id ?? '')
  }, [course])

  if (error) return <Alert>{error}</Alert>
  if (!course) return <Loading />

  async function save() {
    const c = await action.run(
      () =>
        api.put<Course>(`/admin/courses/${id}`, {
          category_id: categoryId,
          license_category_id: licenseId || null,
          sort_order: course!.sort_order,
          translations: cleanTranslations(translations),
        }),
      'Cours enregistré.',
    )
    if (c) setData(c)
  }

  async function changeStatus(to: Status) {
    const c = await action.run(() => api.post<Course>(`/admin/courses/${id}/status`, { status: to }), 'Statut mis à jour.')
    if (c) setData(c)
  }

  async function remove() {
    const ok = await action.run(() => api.del(`/admin/courses/${id}`).then(() => true))
    if (ok) navigate('/courses', { replace: true })
  }

  async function addLesson() {
    const l = await action.run(() =>
      api.post<Lesson>(`/admin/courses/${id}/lessons`, {
        sort_order: (course!.lessons?.length ?? 0) + 1,
        translations: { fr: { title: `Leçon ${(course!.lessons?.length ?? 0) + 1}`, audio_media_id: null } },
      }),
    )
    if (l) navigate(`/lessons/${l.id}`)
  }

  return (
    <>
      <PageHead
        title={course.translations.fr?.title ?? 'Cours'}
        actions={
          <>
            <StatusBadge status={course.status} />
            <button className="btn ghost" onClick={() => navigate('/courses')}>
              <ArrowLeft size={16} /> Retour
            </button>
          </>
        }
      />
      <div className="stack" style={{ gap: 16 }}>
        <div className="card stack">
          <div className="grid cols-2">
            <Field label="Thème">
              <select className="select" value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
                {categories.data?.items.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.translations.fr?.title ?? c.code}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Permis concerné">
              <select className="select" value={licenseId} onChange={(e) => setLicenseId(e.target.value)}>
                <option value="">Tous les permis</option>
                {licenses.data?.items.map((l) => (
                  <option key={l.id} value={l.id}>
                    Permis {l.code}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <TranslationFields value={translations} onChange={setTranslations} />
          <Alert>{action.error}</Alert>
          <Alert kind="success">{action.success}</Alert>
          <div className="row">
            <button className="btn" disabled={action.busy} onClick={() => void save()}>
              Enregistrer
            </button>
            <StatusActions status={course.status} busy={action.busy} onChange={(to) => void changeStatus(to)} />
            <span style={{ flex: 1 }} />
            <DeleteButton what="ce cours et toutes ses leçons" busy={action.busy} onConfirm={() => void remove()} />
          </div>
        </div>

        <div className="card stack">
          <div className="row between">
            <h2>Leçons</h2>
            <button className="btn secondary small" disabled={action.busy} onClick={() => void addLesson()}>
              <Plus size={16} strokeWidth={2.4} /> Nouvelle leçon
            </button>
          </div>
          {course.lessons?.length ? (
            <table className="table">
              <tbody>
                {course.lessons.map((l, i) => (
                  <tr key={l.id} className="clickable" onClick={() => navigate(`/lessons/${l.id}`)}>
                    <td className="muted" style={{ width: 40 }}>
                      {i + 1}
                    </td>
                    <td>
                      <b>{l.translations.fr?.title}</b>
                      {!l.translations.wo?.audio_media_id && <div className="muted small">Audio wolof manquant</div>}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <StatusBadge status={l.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="empty">Aucune leçon. Ajoutez-en une pour écrire le contenu du cours.</div>
          )}
        </div>
      </div>
    </>
  )
}
