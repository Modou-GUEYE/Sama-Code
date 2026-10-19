import { useEffect, useState } from 'react'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Lang, Option, OptionContent, Question, QuestionContent, Status } from '../api/types'
import { MediaField } from '../components/MediaField'
import { DeleteButton } from '../components/DeleteButton'
import { StatusActions } from '../components/StatusActions'
import { Alert, Field, Loading, PageHead, StatusBadge, Tabs, useAction } from '../components/ui'
import { AudioPair } from '../components/AudioPair'

type Draft = Pick<Question, 'question_type' | 'related_lesson_id' | 'contents' | 'options'>

const CODES = ['A', 'B', 'C', 'D', 'E', 'F']

const emptyContent = (): QuestionContent => ({
  text: '',
  audio_media_id: null,
  image_media_id: null,
  explanation: '',
  explanation_audio_media_id: null,
})

const emptyOption = (code: string): Option => ({
  code,
  is_correct: false,
  contents: { fr: { text: '', audio_media_id: null, image_media_id: null }, wo: { text: '', audio_media_id: null, image_media_id: null } },
})

const newDraft = (): Draft => ({
  question_type: 'single',
  related_lesson_id: null,
  contents: { fr: emptyContent(), wo: emptyContent() },
  options: [emptyOption('A'), emptyOption('B')],
})

/** Le wolof n'a que des audios (D25) : on n'envoie pas de texte, et la langue est omise sans audio. */
function wolofContent(c: QuestionContent | undefined): QuestionContent | null {
  if (!c?.audio_media_id && !c?.explanation_audio_media_id) return null
  return { ...emptyContent(), explanation: null, audio_media_id: c.audio_media_id, explanation_audio_media_id: c.explanation_audio_media_id }
}

function wolofOption(c: OptionContent | undefined): OptionContent | null {
  return c?.audio_media_id ? { text: '', image_media_id: null, audio_media_id: c.audio_media_id } : null
}

/** Éditeur d'une question : texte en français, audios en français et en wolof (D25). */
export function QuestionEditor() {
  const { id } = useParams()
  const isNew = id === undefined
  const navigate = useNavigate()
  const [draft, setDraft] = useState<Draft | null>(isNew ? newDraft() : null)
  const [meta, setMeta] = useState<{ status: Status; version: number } | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const save = useAction()

  useEffect(() => {
    if (isNew) return
    api
      .get<Question>(`/admin/questions/${id}`)
      .then(apply)
      .catch((e) => setLoadError(e.message))
  }, [id, isNew])

  function apply(q: Question) {
    setDraft({
      question_type: q.question_type,
      related_lesson_id: q.related_lesson_id,
      contents: { fr: { ...emptyContent(), ...q.contents.fr }, wo: { ...emptyContent(), ...q.contents.wo } },
      options: q.options.map((o) => ({ ...emptyOption(o.code), ...o, contents: { ...emptyOption(o.code).contents, ...o.contents } })),
    })
    setMeta({ status: q.status, version: q.version })
  }

  if (loadError) return <Alert>{loadError}</Alert>
  if (!draft) return <Loading />

  const content = (lang: Lang) => draft.contents[lang] ?? emptyContent()
  const setContent = (lang: Lang, patch: Partial<QuestionContent>) =>
    setDraft({ ...draft, contents: { ...draft.contents, [lang]: { ...content(lang), ...patch } } })

  const setOption = (i: number, patch: Partial<Option>) =>
    setDraft({ ...draft, options: draft.options.map((o, j) => (j === i ? { ...o, ...patch } : o)) })

  const setOptionContent = (i: number, l: Lang, patch: Partial<OptionContent>) => {
    const o = draft.options[i]
    setOption(i, { contents: { ...o.contents, [l]: { text: '', audio_media_id: null, image_media_id: null, ...o.contents[l], ...patch } } })
  }

  const toggleCorrect = (i: number) => {
    if (draft.question_type === 'single') {
      setDraft({ ...draft, options: draft.options.map((o, j) => ({ ...o, is_correct: j === i })) })
    } else {
      setOption(i, { is_correct: !draft.options[i].is_correct })
    }
  }

  const addOption = () => {
    const code = CODES.find((c) => !draft.options.some((o) => o.code === c))
    if (code) setDraft({ ...draft, options: [...draft.options, emptyOption(code)] })
  }

  const removeOption = (i: number) => {
    const options = draft.options.filter((_, j) => j !== i).map((o, j) => ({ ...o, code: CODES[j] }))
    setDraft({ ...draft, options })
  }

  async function submit() {
    if (!draft) return
    const wo = wolofContent(draft.contents.wo)
    const body = {
      ...draft,
      contents: { fr: draft.contents.fr, ...(wo && { wo }) },
      options: draft.options.map((o) => {
        const owo = wolofOption(o.contents.wo)
        return { ...o, contents: { fr: o.contents.fr, ...(owo && { wo: owo }) } }
      }),
    }
    const q = await save.run(
      () => (isNew ? api.post<Question>('/admin/questions', body) : api.put<Question>(`/admin/questions/${id}`, body)),
      'Question enregistrée.',
    )
    if (!q) return
    if (isNew) navigate(`/questions/${q.id}`, { replace: true })
    else apply(q)
  }

  async function changeStatus(to: Status) {
    const q = await save.run(() => api.post<Question>(`/admin/questions/${id}/status`, { status: to }), 'Statut mis à jour.')
    if (q) apply(q)
  }

  async function remove() {
    const ok = await save.run(() => api.del(`/admin/questions/${id}`).then(() => true))
    if (ok) navigate('/questions', { replace: true })
  }

  const correctCount = draft.options.filter((o) => o.is_correct).length

  return (
    <>
      <PageHead
        title={isNew ? 'Nouvelle question' : 'Question'}
        subtitle={meta ? `Version ${meta.version}` : 'Texte en français. Le wolof est uniquement audio.'}
        actions={
          <>
            {meta && <StatusBadge status={meta.status} />}
            <button className="btn ghost" onClick={() => navigate('/questions')}>
              <ArrowLeft size={16} /> Retour
            </button>
          </>
        }
      />

      <div className="stack" style={{ gap: 16 }}>
        <div className="card stack">
          <div className="row between">
            <h2>Énoncé</h2>
            <Tabs<'single' | 'multiple'>
              value={draft.question_type}
              onChange={(t) =>
                setDraft({
                  ...draft,
                  question_type: t,
                  options: t === 'single' && correctCount > 1 ? draft.options.map((o) => ({ ...o, is_correct: false })) : draft.options,
                })
              }
              options={[
                { value: 'single', label: 'Réponse unique' },
                { value: 'multiple', label: 'Réponses multiples' },
              ]}
            />
          </div>

          <textarea
            className="textarea"
            aria-label="Énoncé"
            value={content('fr').text}
            onChange={(e) => setContent('fr', { text: e.target.value })}
            placeholder="Que signifie ce panneau ?"
          />
          <Field label="Illustration">
            <MediaField kind="image" value={content('fr').image_media_id} onChange={(v) => setContent('fr', { image_media_id: v })} />
          </Field>
          <AudioPair
            what="l’énoncé"
            fr={content('fr').audio_media_id}
            wo={content('wo').audio_media_id}
            onChange={(lang, v) => setContent(lang, { audio_media_id: v })}
          />
          <Field label="Explication (affichée après correction)">
            <textarea
              className="textarea"
              style={{ minHeight: 70 }}
              value={content('fr').explanation ?? ''}
              onChange={(e) => setContent('fr', { explanation: e.target.value })}
            />
          </Field>
          <AudioPair
            what="l’explication"
            fr={content('fr').explanation_audio_media_id}
            wo={content('wo').explanation_audio_media_id}
            onChange={(lang, v) => setContent(lang, { explanation_audio_media_id: v })}
          />
        </div>

        <div className="card stack">
          <div className="row between">
            <h2>Propositions</h2>
            <span className="muted small">
              {draft.question_type === 'single'
                ? 'Cochez la seule bonne réponse.'
                : 'Cochez toutes les bonnes réponses : la question n’est juste que si le candidat les coche toutes, sans erreur.'}
            </span>
          </div>
          {draft.options.map((o, i) => (
            <div key={o.code} className={`option-row ${o.is_correct ? 'correct' : ''}`}>
              <div className="option-code">{o.code}</div>
              <input
                className="input"
                style={{ background: '#fff' }}
                placeholder="Texte de la proposition"
                value={o.contents.fr?.text ?? ''}
                onChange={(e) => setOptionContent(i, 'fr', { text: e.target.value })}
              />
              <label className="check">
                <input
                  type={draft.question_type === 'single' ? 'radio' : 'checkbox'}
                  name="correct"
                  checked={o.is_correct}
                  onChange={() => toggleCorrect(i)}
                />
                Bonne réponse
              </label>
              <button
                type="button"
                className="btn ghost small icon-only"
                title="Supprimer"
                disabled={draft.options.length <= 2}
                onClick={() => removeOption(i)}
              >
                <Trash2 size={16} />
              </button>
              <div className="option-audios">
                <AudioPair
                  compact
                  fr={o.contents.fr?.audio_media_id ?? null}
                  wo={o.contents.wo?.audio_media_id ?? null}
                  onChange={(lang, v) => setOptionContent(i, lang, { audio_media_id: v })}
                />
              </div>
            </div>
          ))}
          {draft.options.length < CODES.length && (
            <button type="button" className="btn secondary small" style={{ alignSelf: 'flex-start' }} onClick={addOption}>
              <Plus size={15} strokeWidth={2.4} /> Ajouter une proposition
            </button>
          )}
        </div>

        <Alert>{save.error}</Alert>
        <Alert kind="success">{save.success}</Alert>
        <div className="row">
          <button className="btn" disabled={save.busy} onClick={() => void submit()}>
            {save.busy ? 'Enregistrement…' : 'Enregistrer'}
          </button>
          {meta && !isNew && <StatusActions status={meta.status} busy={save.busy} onChange={(to) => void changeStatus(to)} />}
          {meta?.status === 'published' && (
            <span className="muted small">Modifier une question publiée la repasse en brouillon (nouvelle version).</span>
          )}
          {!isNew && (
            <>
              <span style={{ flex: 1 }} />
              <DeleteButton what="cette question" busy={save.busy} onConfirm={() => void remove()} />
            </>
          )}
        </div>
      </div>
    </>
  )
}
