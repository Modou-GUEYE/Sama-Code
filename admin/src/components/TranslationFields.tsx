import type { Lang, Translation, Translations } from '../api/types'
import { AudioPair } from './AudioPair'
import { Field } from './ui'

/** Titre en français, avec son audio en français et en wolof (D25 : le wolof est uniquement audio). */
export function TranslationFields({
  value,
  onChange,
  titleLabel = 'Titre',
  withDescription = false,
}: {
  value: Translations
  onChange: (v: Translations) => void
  titleLabel?: string
  withDescription?: boolean
}) {
  const set = (lang: Lang, patch: Partial<Translation>) =>
    onChange({ ...value, [lang]: { title: '', audio_media_id: null, ...value[lang], ...patch } })

  return (
    <div className="stack">
      <Field label={titleLabel}>
        <input className="input" value={value.fr?.title ?? ''} onChange={(e) => set('fr', { title: e.target.value })} />
      </Field>
      {withDescription && (
        <Field label="Description">
          <textarea
            className="textarea"
            style={{ minHeight: 60 }}
            value={value.fr?.description ?? ''}
            onChange={(e) => set('fr', { description: e.target.value })}
          />
        </Field>
      )}
      <AudioPair
        what={titleLabel.toLowerCase() === 'nom' ? 'nom' : 'titre'}
        fr={value.fr?.audio_media_id ?? null}
        wo={value.wo?.audio_media_id ?? null}
        onChange={(lang, v) => set(lang, { audio_media_id: v })}
      />
    </div>
  )
}

/** Garde le français s'il a un titre, et le wolof seulement s'il a un audio (sans texte). */
export function cleanTranslations(t: Translations): Translations {
  const out: Translations = {}
  if (t.fr && t.fr.title.trim() !== '') out.fr = t.fr
  if (t.wo?.audio_media_id) out.wo = { title: '', audio_media_id: t.wo.audio_media_id }
  return out
}
