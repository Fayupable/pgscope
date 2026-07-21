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

export function startRecording(minutes: number): Promise<void> {
    return postJSON(`/api/v1/record/start?minutes=${minutes}`)
}

export function stopRecording(): Promise<void> {
    return postJSON('/api/v1/record/stop')
}

export async function downloadHistory(): Promise<void> {
    const response = await fetch('/api/v1/history')
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