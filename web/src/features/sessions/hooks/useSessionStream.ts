import { useMonitoringStream } from '../../../shared/api/monitoringStreamContext'
import type { Session } from '../../../shared/types/session'

export function useSessionStream(): Session[] {
    return useMonitoringStream().sessions
}