import { useEffect, useState } from 'react'
import { ArrowDown, ArrowLeft, ArrowUp, Image, Plus, Trash2, Type, Volume2 } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Block, Lang, Lesson, Status, Translations } from '../api/types'
import { MediaField } from '../components/MediaField'
import { DeleteButton } from '../components/DeleteButton'
import { StatusActions } from '../components/StatusActions'
import { cleanTranslations, TranslationFields } from '../components/TranslationFields'
import { Alert, Loading, PageHead, StatusBadge, Tabs, useAction, useLoad } from '../components/ui'

const BLOCK = {
  text: { label: 'Texte', Icon: Type },
  audio: { label: 'Audio', Icon: Volume2 },
  image: { label: 'Image', Icon: Image },
}

/** Leçon : titre et blocs en français (texte, audio, image) ; en wolof, des audios seulement (D25). */
export function LessonEditor() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { data: lesson, error, setData } = useLoad(() => api.get<Lesson>(`/admin/lessons/${id}`), [id])
  const [translations, setTranslations] = useState<Translations>({})
  const [blocks, setBlocks] = useState<Record<Lang, Block[]>>({ fr: [], wo: [] })
  const [lang, setLang] = useState<Lang>('fr')
  const action = useAction()

  useEffect(() => {
    if (!lesson) return
    setTranslations(lesson.translations)
    setBlocks({ fr: lesson.contents?.fr ?? [], wo: (lesson.contents?.wo ?? []).filter((b) => b.content_type === 'audio') })
  }, [lesson])

  if (error) return <Alert>{error}</Alert>
  if (!lesson) return <Loading />

  const list = blocks[lang]
  const setList = (next: Block[]) => setBlocks({ ...blocks, [lang]: next })
  const patch = (i: number, p: Partial<Block>) => setList(list.map((b, j) => (j === i ? { ...b, ...p } : b)))
  const move = (i: number, d: number) => {
    const j = i + d
    if (j < 0 || j >= list.length) return
    const next = [...list]
    ;[next[i], next[j]] = [next[j], next[i]]
    setList(next)
  }

  async function save() {
    const ok = await action.run(async () => {
      await api.put(`/admin/lessons/${id}`, { sort_order: lesson!.sort_order, translations: cleanTranslations(translations) })
      // Les blocs vides (texte non saisi, média non choisi) ne sont pas envoyés.
      const body = Object.fromEntries(
        (Object.entries(blocks) as [Lang, Block[]][]).map(([l, bs]) => [
          l,
          bs.filter((b) => (b.content_type === 'text' ? (b.text ?? '').trim() !== '' : !!b.media_id)),
        ]),
      )
      return api.put<Lesson>(`/admin/lessons/${id}/contents`, body)
    }, 'Leçon enregistrée.')
    if (ok) setData(ok)
  }

  async function remove() {
    const ok = await action.run(() => api.del(`/admin/lessons/${id}`).then(() => true))
    if (ok) navigate(`/courses/${lesson!.course_id}`, { replace: true })
  }

  async function changeStatus(to: Status) {
    const l = await action.run(() => api.post<Lesson>(`/admin/lessons/${id}/status`, { status: to }), 'Statut mis à jour.')
    if (l) setData(l)
  }

  return (
    <>
      <PageHead
        title={lesson.translations.fr?.title ?? 'Leçon'}
        actions={
          <>
            <StatusBadge status={lesson.status} />
            <button className="btn ghost" onClick={() => navigate(`/courses/${lesson.course_id}`)}>
              <ArrowLeft size={16} /> Retour au cours
            </button>
          </>
        }
      />
      <div className="stack" style={{ gap: 16 }}>
        <div className="card">
          <TranslationFields value={translations} onChange={setTranslations} />
        </div>

        <div className="card stack">
          <div className="row between">
            <Tabs<Lang>
              value={lang}
              onChange={setLang}
              options={[
                { value: 'fr', label: `Leçon en français (${blocks.fr.length})` },
                { value: 'wo', label: `Audios en wolof (${blocks.wo.length})` },
              ]}
            />
            <div className="row">
              {(lang === 'wo' ? (['audio'] as const) : (['text', 'audio', 'image'] as const)).map((t) => (
                <button
                  key={t}
                  className="btn secondary small"
                  onClick={() => setList([...list, { content_type: t, text: t === 'text' ? '' : null, media_id: null }])}
                >
                  {(() => {
                    const { Icon, label } = BLOCK[t]
                    return (
                      <>
                        <Plus size={15} strokeWidth={2.4} /> <Icon size={15} /> {label}
                      </>
                    )
                  })()}
                </button>
              ))}
            </div>
          </div>
          {lang === 'wo' && (
            <div className="muted small">
              Le candidat qui écoute en wolof voit le texte et les images en français, et entend ces audios à la place des audios
              français.
            </div>
          )}
          {list.length === 0 && (
            <div className="empty">
              {lang === 'wo'
                ? 'Ajoutez l’enregistrement de la leçon en wolof.'
                : 'Ajoutez des blocs : texte à lire, audio à écouter, illustration.'}
            </div>
          )}
          {list.map((b, i) => (
            <div key={i} className="block">
              <div className="stack" style={{ gap: 6 }}>
                <span className="small muted">
                  {i + 1}. {BLOCK[b.content_type].label}
                </span>
                {b.content_type === 'text' ? (
                  <textarea
                    className="textarea"
                    style={{ background: '#fff' }}
                    value={b.text ?? ''}
                    onChange={(e) => patch(i, { text: e.target.value })}
                  />
                ) : (
                  <MediaField kind={b.content_type} value={b.media_id} onChange={(v) => patch(i, { media_id: v })} />
                )}
              </div>
              <div className="stack" style={{ gap: 2 }}>
                <button className="btn ghost small icon-only" disabled={i === 0} onClick={() => move(i, -1)} title="Monter">
                  <ArrowUp size={16} />
                </button>
                <button
                  className="btn ghost small icon-only"
                  disabled={i === list.length - 1}
                  onClick={() => move(i, 1)}
                  title="Descendre"
                >
                  <ArrowDown size={16} />
                </button>
                <button className="btn ghost small icon-only" onClick={() => setList(list.filter((_, j) => j !== i))} title="Supprimer">
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>

        <Alert>{action.error}</Alert>
        <Alert kind="success">{action.success}</Alert>
        <div className="row">
          <button className="btn" disabled={action.busy} onClick={() => void save()}>
            Enregistrer
          </button>
          <StatusActions status={lesson.status} busy={action.busy} onChange={(to) => void changeStatus(to)} />
          <span style={{ flex: 1 }} />
          <DeleteButton what="cette leçon" busy={action.busy} onConfirm={() => void remove()} />
        </div>
      </div>
    </>
  )
}
