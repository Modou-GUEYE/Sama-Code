import type { Lang, Translation, Translations } from '../api/types'
import { MediaField } from './MediaField'
import { Field } from './ui'

const LABEL: Record<Lang, string> = { fr: 'Français', wo: 'Wolof' }

/** Titre (et audio du titre) en français et en wolof, côte à côte. */
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
    <div className="grid cols-2">
      {(['fr', 'wo'] as Lang[]).map((lang) => (
        <div key={lang} className="stack">
          <Field label={`${titleLabel} — ${LABEL[lang]}`}>
            <input className="input" value={value[lang]?.title ?? ''} onChange={(e) => set(lang, { title: e.target.value })} />
          </Field>
          {withDescription && (
            <Field label={`Description — ${LABEL[lang]}`}>
              <textarea
                className="textarea"
                style={{ minHeight: 60 }}
                value={value[lang]?.description ?? ''}
                onChange={(e) => set(lang, { description: e.target.value })}
              />
            </Field>
          )}
          <MediaField
            kind="audio"
            label={`Audio du titre (${LABEL[lang].toLowerCase()})`}
            value={value[lang]?.audio_media_id}
            onChange={(v) => set(lang, { audio_media_id: v })}
          />
        </div>
      ))}
    </div>
  )
}

/** Supprime les langues sans titre avant l'envoi. */
export function cleanTranslations(t: Translations): Translations {
  return Object.fromEntries(Object.entries(t).filter(([, v]) => v && v.title.trim() !== '')) as Translations
}
