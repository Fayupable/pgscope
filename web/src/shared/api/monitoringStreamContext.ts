import { createContext, useContext } from 'react'
import type { Session, DatabaseActivityStats } from '../types/session'

export interface MonitoringStreamState {
  sessions: Session[]
  dbStats: DatabaseActivityStats
}

export const MonitoringStreamContext = createContext<MonitoringStreamState | null>(null)

export function useMonitoringStream(): MonitoringStreamState {
  const context = useContext(MonitoringStreamContext)
  if (!context) {
    throw new Error('useMonitoringStream must be used within a MonitoringStreamProvider')
  }
  return context
}