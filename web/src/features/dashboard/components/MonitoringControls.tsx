import { useState } from 'react'
import { startMonitoring, stopMonitoring, startRecording, stopRecording, downloadHistory } from '../../../shared/api/controlClient'
import './MonitoringControls.css'

const MONITOR_OPTIONS = [
  { label: '5 min', value: 5 },
  { label: '10 min', value: 10 },
  { label: '30 min', value: 30 },
  { label: 'Full', value: 0 },
]

const RECORD_OPTIONS = [
  { label: '5 min', value: 5 },
  { label: '10 min', value: 10 },
  { label: '15 min', value: 15 },
  { label: '30 min', value: 30 },
]

export function MonitoringControls() {
  const [monitorMinutes, setMonitorMinutes] = useState(MONITOR_OPTIONS[0].value)
  const [recordMinutes, setRecordMinutes] = useState(RECORD_OPTIONS[0].value)
  const [isMonitoring, setIsMonitoring] = useState(false)
  const [isRecording, setIsRecording] = useState(false)
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
          <button className="monitoring-controls__stop monitoring-controls__stop--live" onClick={() => handleAction(stopMonitoring, () => { setIsMonitoring(false); setIsRecording(false) })}>
            ● Live
          </button>
        )}
      </div>

      <div className="monitoring-controls__group">
        <span className="monitoring-controls__label">Record</span>
        <select value={recordMinutes} onChange={(e) => setRecordMinutes(Number(e.target.value))} disabled={isRecording}>
          {RECORD_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
        {!isRecording ? (
          <button className="monitoring-controls__start" onClick={() => handleAction(() => startRecording(recordMinutes), () => setIsRecording(true))} disabled={!isMonitoring}>
            Start
          </button>
        ) : (
          <button className="monitoring-controls__stop monitoring-controls__stop--recording" onClick={() => handleAction(stopRecording, () => setIsRecording(false))}>
            ● Recording
          </button>
        )}
        <button className="monitoring-controls__download" onClick={() => handleAction(downloadHistory, () => {})}>
          Download JSON
        </button>
      </div>

      {error && <p className="monitoring-controls__error">{error}</p>}
    </div>
  )
}