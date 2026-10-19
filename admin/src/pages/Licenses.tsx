import { useState } from 'react'
import { api } from '../api/client'
import type { License } from '../api/types'
import { Alert, Field, Loading, PageHead, useAction, useLoad } from '../components/ui'

/** Paramètres des permis : barème et durée d'examen (administrateur uniquement). */
export function Licenses() {
  const { data, error, loading } = useLoad(() => api.get<{ items: License[] }>('/admin/license-categories'), [])
  if (loading && !data) return <Loading />
  return (
    <>
      <PageHead
        title="Permis"
        subtitle="Les changements s’appliquent aux prochains examens ; les séries déjà publiées gardent leur version."
      />
      <Alert>{error}</Alert>
      <div className="grid cols-2">
        {data?.items.map((l) => <LicenseCard key={l.id} license={l} />)}
      </div>
    </>
  )
}

function LicenseCard({ license }: { license: License }) {
  const [form, setForm] = useState(license)
  const action = useAction()
  const num = (key: keyof License) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm({ ...form, [key]: Number(e.target.value) })

  async function save() {
    const out = await action.run(
      () =>
        api.patch<License>(`/admin/license-categories/${license.id}`, {
          name: form.name,
          series_count: form.series_count,
          questions_per_series: form.questions_per_series,
          max_faults: form.max_faults,
          exam_duration_minutes: form.exam_duration_minutes,
          is_active: form.is_active,
        }),
      'Paramètres enregistrés.',
    )
    if (out) setForm(out)
  }

  return (
    <div className="card stack">
      <div className="row between">
        <h2>Permis {license.code}</h2>
        <label className="check" style={{ paddingTop: 0 }}>
          <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
          Actif
        </label>
      </div>
      <Field label="Nom">
        <input className="input" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
      </Field>
      <div className="grid cols-2">
        <Field label="Nombre de séries">
          <input className="input" type="number" min={1} value={form.series_count} onChange={num('series_count')} />
        </Field>
        <Field label="Questions par série">
          <input className="input" type="number" min={1} value={form.questions_per_series} onChange={num('questions_per_series')} />
        </Field>
        <Field label="Fautes autorisées (réussi si ≤)">
          <input className="input" type="number" min={0} value={form.max_faults} onChange={num('max_faults')} />
        </Field>
        <Field label="Durée de l’examen (minutes)">
          <input className="input" type="number" min={1} value={form.exam_duration_minutes} onChange={num('exam_duration_minutes')} />
        </Field>
      </div>
      <Alert>{action.error}</Alert>
      <Alert kind="success">{action.success}</Alert>
      <button className="btn" style={{ alignSelf: 'flex-start' }} disabled={action.busy} onClick={() => void save()}>
        Enregistrer
      </button>
    </div>
  )
}
