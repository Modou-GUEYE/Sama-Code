import { Trash2 } from 'lucide-react'

/** Bouton de suppression définitive, avec confirmation. Les règles (droits, historique
 *  des candidats) sont vérifiées par l'API, qui explique un éventuel refus. */
export function DeleteButton({
  what,
  busy,
  onConfirm,
}: {
  what: string
  busy?: boolean
  onConfirm: () => void
}) {
  return (
    <button
      type="button"
      className="btn danger"
      disabled={busy}
      onClick={() => {
        if (confirm(`Supprimer définitivement ${what} ? Cette action est irréversible.`)) onConfirm()
      }}
    >
      <Trash2 size={16} /> Supprimer
    </button>
  )
}
