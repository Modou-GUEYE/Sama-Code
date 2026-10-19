import { useRef, useState } from 'react'
import { api, errorMessage } from '../api/client'
import type { MediaAsset } from '../api/types'

const ACCEPT = {
  image: 'image/png,image/jpeg,image/webp',
  audio: 'audio/mpeg,audio/mp4,audio/x-m4a,audio/ogg,audio/wav',
}

export function mediaUrl(id: string) {
  return `/api/v1/media/${id}/file`
}

/** Choix d'un fichier audio ou image : envoi, aperçu (image ou lecteur), retrait. */
export function MediaField({
  kind,
  value,
  onChange,
  label,
}: {
  kind: 'image' | 'audio'
  value: string | null | undefined
  onChange: (id: string | null) => void
  label?: string
}) {
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function upload(file: File) {
    setBusy(true)
    setError(null)
    try {
      const form = new FormData()
      form.append('file', file)
      const asset = await api.post<MediaAsset>('/admin/media', form)
      if (asset.media_type !== kind) {
        setError(kind === 'image' ? 'Ce fichier n’est pas une image.' : 'Ce fichier n’est pas un audio.')
        return
      }
      onChange(asset.id)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
      if (input.current) input.current.value = ''
    }
  }

  return (
    <div className="stack" style={{ gap: 4 }}>
      <div className="media-field">
        <span style={{ width: 22, textAlign: 'center' }}>{kind === 'image' ? '🖼️' : '🔊'}</span>
        {value ? (
          kind === 'image' ? (
            <img src={mediaUrl(value)} alt="" />
          ) : (
            <audio controls preload="none" src={mediaUrl(value)} />
          )
        ) : (
          <span className="muted small" style={{ flex: 1 }}>
            {label ?? (kind === 'image' ? 'Aucune image' : 'Aucun audio')}
          </span>
        )}
        <span style={{ flex: value ? 1 : 0 }} />
        <button type="button" className="btn secondary small" disabled={busy} onClick={() => input.current?.click()}>
          {busy ? 'Envoi…' : value ? 'Remplacer' : 'Ajouter'}
        </button>
        {value && (
          <button type="button" className="btn ghost small icon-only" title="Retirer" onClick={() => onChange(null)}>
            ✕
          </button>
        )}
        <input
          ref={input}
          type="file"
          hidden
          accept={ACCEPT[kind]}
          onChange={(e) => e.target.files?.[0] && upload(e.target.files[0])}
        />
      </div>
      {error && <span className="small" style={{ color: 'var(--red)' }}>{error}</span>}
    </div>
  )
}
