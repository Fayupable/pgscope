import type { Session } from './session'

export type SnapshotTrigger = 'periodic' | 'incident'

export interface Snapshot {
    sessions: Session[]
    capturedAt: string
    trigger: SnapshotTrigger
}