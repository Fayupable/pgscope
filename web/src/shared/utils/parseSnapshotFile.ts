import type { Snapshot } from '../types/snapshot'

export async function parseSnapshotFile(file: File): Promise<Snapshot[]> {
    const text = await file.text()

    let parsed: unknown
    try {
        parsed = JSON.parse(text)
    } catch {
        throw new Error('File is not valid JSON')
    }

    if (!Array.isArray(parsed)) {
        throw new Error('Expected a JSON array of snapshots')
    }

    return parsed.map(validateSnapshot)
}

function validateSnapshot(value: unknown, index: number): Snapshot {
    if (typeof value !== 'object' || value === null) {
        throw new Error(`Entry ${index} is not an object`)
    }

    const candidate = value as Record<string, unknown>

    if (!Array.isArray(candidate.sessions)) {
        throw new Error(`Entry ${index} is missing "sessions" array`)
    }
    if (typeof candidate.capturedAt !== 'string') {
        throw new Error(`Entry ${index} is missing "capturedAt" string`)
    }

    return candidate as unknown as Snapshot
}