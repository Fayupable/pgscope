import { useCallback, useEffect, useState } from 'react'
import { getInsights } from '../../../shared/api/insightsClient'
import type { Insights } from '../../../shared/types/insights'

export function useInsights() {
    const [insights, setInsights] = useState<Insights | null>(null)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const refresh = useCallback(async () => {
        try {
            setLoading(true)
            setError(null)
            setInsights(await getInsights())
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Request failed')
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        refresh()
    }, [refresh])

    return { insights, loading, error, refresh }
}