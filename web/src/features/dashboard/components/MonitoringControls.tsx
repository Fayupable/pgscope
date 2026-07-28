import { useState } from 'react'
import { startMonitoring, stopMonitoring, downloadHistory } from '../../../shared/api/controlClient'
import type { HistoryWindow } from '../../../shared/api/controlClient'
import './MonitoringControls.css'

const MONITOR_OPTIONS = [
  { label: '5 min', value: 5 },
  { label: '10 min', value: 10 },
  { label: '30 min', value: 30 },
  { label: 'Full', value: 0 },
]

const WINDOW_OPTIONS: { label: string; value: HistoryWindow }[] = [
  { label: '1 hour', value: '1h' },
  { label: '3 hours', value: '3h' },
  { label: '6 hours', value: '6h' },
  { label: '12 hours', value: '12h' },
  { label: '24 hours', value: '24h' },
]

export function MonitoringControls() {
  const [monitorMinutes, setMonitorMinutes] = useState(MONITOR_OPTIONS[0].value)
  const [historyWindow, setHistoryWindow] = useState<HistoryWindow>(WINDOW_OPTIONS[0].value)
  const [isMonitoring, setIsMonitoring] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleAction(action: () => Promise<void>, onSuccess: () => void) {
    try {
      setError(null)
      await action()
      onSuccess()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed')
    }
  }

  return (
    <div className="monitoring-controls">
      <div className="monitoring-controls__group">
        <span className="monitoring-controls__label">Monitor</span>
        <select value={monitorMinutes} onChange={(e) => setMonitorMinutes(Number(e.target.value))} disabled={isMonitoring}>
          {MONITOR_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        {!isMonitoring ? (
          <button className="monitoring-controls__start" onClick={() => handleAction(() => startMonitoring(monitorMinutes), () => setIsMonitoring(true))}>
            Start
          </button>
        ) : (
          <button className="monitoring-controls__stop monitoring-controls__stop--live" onClick={() => handleAction(stopMonitoring, () => setIsMonitoring(false))}>
            ● Live
          </button>
        )}
      </div>

      <div className="monitoring-controls__group">
        <span className="monitoring-controls__label">History</span>
        <select value={historyWindow} onChange={(e) => setHistoryWindow(e.target.value as HistoryWindow)}>
          {WINDOW_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        <button className="monitoring-controls__download" onClick={() => handleAction(() => downloadHistory(historyWindow), () => {})}>
          Download JSON
        </button>
      </div>

      {error && <p className="monitoring-controls__error">{error}</p>}
    </div>
  )
}