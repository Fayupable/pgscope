import type { Insights } from '../types/insights'

export async function getInsights(): Promise<Insights> {
    const response = await fetch('/api/v1/insights')
    if (!response.ok) {
        if (response.status === 429) {
            throw new Error('Rate limited — please wait a few seconds before refreshing again.')
        }
        throw new Error(`Failed to fetch insights: ${response.status}`)
    }
    return response.json()
}