import './GraphLegend.css'

/**
 * A small, always-visible reference panel explaining the graph's visual
 * language — node status colors, edge types, and cluster shading — so a
 * viewer doesn't have to guess or click around to understand what they're
 * looking at.
 */
export function GraphLegend() {
  return (
    <div className="graph-legend">
      <div className="graph-legend__section">
        <span className="graph-legend__title">Session status</span>
        <div className="graph-legend__row">
          <span className="graph-legend__dot" style={{ borderColor: 'var(--status-success)' }} />
          Active
        </div>
        <div className="graph-legend__row">
          <span className="graph-legend__dot" style={{ borderColor: 'var(--status-warning)' }} />
          Long-running / idle in transaction
        </div>
        <div className="graph-legend__row">
          <span className="graph-legend__dot" style={{ borderColor: 'var(--status-danger)' }} />
          Blocked
        </div>
        <div className="graph-legend__row">
          <span className="graph-legend__dot" style={{ borderColor: 'var(--status-neutral)' }} />
          Idle
        </div>
      </div>

      <div className="graph-legend__section">
        <span className="graph-legend__title">Connections</span>
        <div className="graph-legend__row">
          <span className="graph-legend__line graph-legend__line--blocking" />
          Waiting on (blocking)
        </div>
        <div className="graph-legend__row">
          <span className="graph-legend__line graph-legend__line--bridge" />
          Cross-cluster link
        </div>
        <div className="graph-legend__row">
          <span className="graph-legend__swatch" />
          Cluster group
        </div>
      </div>
    </div>
  )
}
