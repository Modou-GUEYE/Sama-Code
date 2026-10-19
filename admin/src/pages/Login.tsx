import { useState, type FormEvent } from 'react'
import { errorMessage } from '../api/client'
import { useAuth } from '../auth/AuthContext'
import { Alert, Field } from '../components/ui'

/** Connexion avec le même compte que l'application mobile : téléphone + code secret. */
export function Login() {
  const { login } = useAuth()
  const [phone, setPhone] = useState('')
  const [pin, setPin] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await login(phone, pin)
    } catch (err) {
      setError(errorMessage(err))
      setPin('')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login">
      <form className="card stack" onSubmit={submit}>
        <img src="/logo.png" alt="Sama Code" />
        <div style={{ textAlign: 'center' }}>
          <h1>Administration</h1>
          <p className="muted" style={{ margin: '4px 0 0' }}>Connectez-vous avec votre compte Sama Code.</p>
        </div>
        <Field label="Numéro de téléphone">
          <input
            className="input"
            inputMode="tel"
            autoComplete="tel"
            placeholder="77 123 45 67"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            required
          />
        </Field>
        <Field label="Code secret (4 chiffres)">
          <input
            className="input"
            type="password"
            inputMode="numeric"
            autoComplete="current-password"
            maxLength={4}
            value={pin}
            onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))}
            required
          />
        </Field>
        <Alert>{error}</Alert>
        <button className="btn" disabled={busy || pin.length !== 4}>
          {busy ? 'Connexion…' : 'Se connecter'}
        </button>
      </form>
    </div>
  )
}
