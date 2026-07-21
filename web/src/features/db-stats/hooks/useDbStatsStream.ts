import { useMonitoringStream } from '../../../shared/api/monitoringStreamContext'
import type { DatabaseActivityStats } from '../../../shared/types/session'

export function useDbStatsStream(): DatabaseActivityStats {
  return useMonitoringStream().dbStats
}