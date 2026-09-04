import { useEffect, useState } from 'react'
import { checkAuthStatus, logout } from './shared/api/authClient'
import { LoginScreen } from './features/auth/components/LoginScreen'
import { MonitoringStreamProvider } from './shared/api/MonitoringStreamProvider'
import { EngineProvider } from './shared/api/EngineProvider'
import { Dashboard } from './features/dashboard/components/Dashboard'
import { useTheme } from './shared/hooks/useTheme'

type AuthState = 'checking' | 'authenticated' | 'unauthenticated'

function App() {
  const [authState, setAuthState] = useState<AuthState>('checking')
  const { theme, toggleTheme } = useTheme()

  useEffect(() => {
    checkAuthStatus().then((ok) => setAuthState(ok ? 'authenticated' : 'unauthenticated'))
  }, [])

  async function handleLogout() {
    await logout()
    setAuthState('unauthenticated')
  }

  if (authState === 'checking') {
    return null
  }

  if (authState === 'unauthenticated') {
    return <LoginScreen onSuccess={() => setAuthState('authenticated')} />
  }

  return (
    <EngineProvider>
      <MonitoringStreamProvider>
        <Dashboard onLogout={handleLogout} theme={theme} onToggleTheme={toggleTheme} />
      </MonitoringStreamProvider>
    </EngineProvider>
  )
}

export default App
