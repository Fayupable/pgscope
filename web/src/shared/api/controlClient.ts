async function postJSON(url: string): Promise<void> {
    const response = await fetch(url, { method: 'POST' })
    if (!response.ok) {
        const message = await response.text()
        throw new Error(message || `Request failed: ${response.status}`)
    }
}

export function startMonitoring(minutes: number): Promise<void> {
    return postJSON(`/api/v1/monitor/start?minutes=${minutes}`)
}

export function stopMonitoring(): Promise<void> {
    return postJSON('/api/v1/monitor/stop')
}

export type HistoryWindow = '1h' | '3h' | '6h' | '12h' | '24h'

// Recording is no longer a separate start/stop action — it happens
// automatically in the background whenever monitoring is active. This just
// downloads whatever's already been recorded for the requested window.
export async function downloadHistory(window: HistoryWindow): Promise<void> {
    const response = await fetch(`/api/v1/history?window=${window}`)
    if (!response.ok) {
        throw new Error(`Failed to fetch history: ${response.status}`)
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)

    const link = document.createElement('a')
    link.href = url
    link.download = `pgscope-history-${Date.now()}.json`
    link.click()

    URL.revokeObjectURL(url)
}