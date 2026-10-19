import type { Lang } from '../api/types'
import { MediaField } from './MediaField'
import { Field } from './ui'

/** Audio français et audio wolof d'un même texte, côte à côte (D25 : le wolof est uniquement audio). */
export function AudioPair({
  what,
  fr,
  wo,
  onChange,
  compact = false,
}: {
  what?: string
  fr: string | null
  wo: string | null
  onChange: (lang: Lang, value: string | null) => void
  compact?: boolean
}) {
  const field = (lang: Lang, value: string | null) => {
    const name = lang === 'fr' ? 'français' : 'wolof'
    const media = <MediaField kind="audio" label={`Aucun audio ${name}`} value={value} onChange={(v) => onChange(lang, v)} />
    if (!compact) return <Field label={`Audio ${name} de ${what}`}>{media}</Field>
    return (
      <div className="stack" style={{ gap: 4 }}>
        <span className="small muted">Audio {name}</span>
        {media}
      </div>
    )
  }
  return (
    <div className="grid cols-2" style={compact ? { gap: 10 } : undefined}>
      {field('fr', fr)}
      {field('wo', wo)}
    </div>
  )
}
