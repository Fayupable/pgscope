import { useState } from 'react'
import { login } from '../../../shared/api/authClient'
import './LoginScreen.css'

interface LoginScreenProps {
    onSuccess: () => void
}

export function LoginScreen({ onSuccess }: LoginScreenProps) {
    const [key, setKey] = useState('')
    const [error, setError] = useState<string | null>(null)
    const [submitting, setSubmitting] = useState(false)

    async function handleSubmit(event: React.FormEvent) {
        event.preventDefault()
        setSubmitting(true)
        setError(null)
        try {
            await login(key)
            onSuccess()
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Login failed')
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <div className="login-screen">
            <form className="login-screen__card" onSubmit={handleSubmit}>
                <h1>pgscope</h1>
                <p className="login-screen__subtitle">Enter the API key to continue</p>
                <input
                    type="password"
                    value={key}
                    onChange={(e) => setKey(e.target.value)}
                    placeholder="API key"
                    autoFocus
                />
                <button type="submit" disabled={submitting || key.length === 0}>
                    {submitting ? 'Checking…' : 'Continue'}
                </button>
                {error && <p className="login-screen__error">{error}</p>}
            </form>
        </div>
    )
}