export function formatDuration(ms: number): string {
    if (ms < 1000) {
        return `${ms.toFixed(1)} ms`
    }

    const seconds = ms / 1000
    if (seconds < 60) {
        return `${seconds.toFixed(2)} s`
    }

    const minutes = seconds / 60
    if (minutes < 60) {
        return `${minutes.toFixed(2)} min`
    }

    const hours = minutes / 60
    if (hours < 24) {
        return `${hours.toFixed(2)} h`
    }

    const days = hours / 24
    return `${days.toFixed(2)} d`
}