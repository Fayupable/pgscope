async function postJSON(url: string, body?: unknown): Promise<Response> {
    return fetch(url, {
        method: 'POST',
        credentials: 'include',
        headers: body ? { 'Content-Type': 'application/json' } : undefined,
        body: body ? JSON.stringify(body) : undefined,
    })
}

export async function login(key: string): Promise<void> {
    const response = await postJSON('/api/v1/auth/login', { key })
    if (!response.ok) {
        throw new Error(response.status === 401 ? 'Invalid key' : `Login failed: ${response.status}`)
    }
}

export async function logout(): Promise<void> {
    await postJSON('/api/v1/auth/logout')
}

export async function checkAuthStatus(): Promise<boolean> {
    const response = await fetch('/api/v1/auth/status', { credentials: 'include' })
    return response.ok
}