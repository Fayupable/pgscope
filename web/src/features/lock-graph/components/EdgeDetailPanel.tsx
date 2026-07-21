import type { Session } from '../../../shared/types/session'
import './EdgeDetailPanel.css'

interface EdgeDetailPanelProps {
  blocker: Session
  blocked: Session
  position: { x: number; y: number }
  onClose: () => void
}

/**
 * Shown when clicking an edge in the graph — explains the specific
 * blocking relationship it represents (who is waiting on whom, on what
 * kind of lock, for how long) rather than requiring the viewer to infer
 * it from two separate node detail panels.
 */
export function EdgeDetailPanel({ blocker, blocked, position, onClose }: EdgeDetailPanelProps) {
  return (
    <div className="edge-detail-panel" style={{ left: position.x, top: position.y }}>
      <div className="edge-detail-panel__header">
        <span className="edge-detail-panel__title">Blocking relationship</span>
        <button className="edge-detail-panel__close" onClick={onClose}>
          ✕
        </button>
      </div>

      <p className="edge-detail-panel__summary">
        <span className="edge-detail-panel__pid">PID {blocked.id}</span> is waiting on{' '}
        <span className="edge-detail-panel__pid">PID {blocker.id}</span>
      </p>

      <dl className="edge-detail-panel__meta">
        <div>
          <dt>Wait event</dt>
          <dd>{blocked.waitEventType === 'none' ? '—' : `${blocked.waitEventType}: ${blocked.waitEvent}`}</dd>
        </div>
        <div>
          <dt>Waiting for</dt>
          <dd>{blocked.durationSeconds.toFixed(1)}s</dd>
        </div>
        <div>
          <dt>Blocker running</dt>
          <dd>{blocker.durationSeconds.toFixed(1)}s</dd>
        </div>
        <div>
          <dt>Blocker state</dt>
          <dd>{blocker.state}</dd>
        </div>
      </dl>

      <p className="edge-detail-panel__query-label">PID {blocker.id} is running:</p>
      <p className="edge-detail-panel__query">{blocker.query}</p>
    </div>
  )
}
