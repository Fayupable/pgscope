import type { Connection } from '../types/connection'

export async function getConnection(): Promise<Connection> {
    const response = await fetch('/api/v1/connection', { credentials: 'include' })
    if (!response.ok) {
        throw new Error(`Failed to fetch connection: ${response.status}`)
    }
    return response.json()
}
