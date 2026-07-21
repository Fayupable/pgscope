import { useViewport } from '@xyflow/react'
import { line, curveCatmullRomClosed } from 'd3-shape'
import type { ClusterHull } from '../../../shared/utils/graph'

const HULL_COLORS = [
  { fill: 'var(--cluster-hull-1)', stroke: 'var(--cluster-hull-stroke-1)' },
  { fill: 'var(--cluster-hull-2)', stroke: 'var(--cluster-hull-stroke-2)' },
  { fill: 'var(--cluster-hull-3)', stroke: 'var(--cluster-hull-stroke-3)' },
  { fill: 'var(--cluster-hull-4)', stroke: 'var(--cluster-hull-stroke-4)' },
  { fill: 'var(--cluster-hull-5)', stroke: 'var(--cluster-hull-stroke-5)' },
]

// Node positions from graphLayout represent each node's top-left corner (to
// match React Flow's own convention), but the hull was computed from those
// same raw coordinates. Offsetting by half the node's rendered size (44px)
// aligns the hull shape with where nodes visually appear (their centers).
const NODE_CENTER_OFFSET = 22

const hullLine = line<{ x: number; y: number }>()
  .x((p) => p.x)
  .y((p) => p.y)
  .curve(curveCatmullRomClosed)

interface ClusterHullsProps {
  hulls: ClusterHull[]
}

/**
 * Renders each cluster's convex hull as a smooth, rounded shaded region
 * behind the graph's nodes/edges (a Catmull-Rom closed curve through the
 * hull points, rather than a sharp-cornered polygon), manually kept in
 * sync with React Flow's pan/zoom via useViewport() — this component must
 * be rendered as a child of <ReactFlow> to access that context.
 */
export function ClusterHulls({ hulls }: ClusterHullsProps) {
  const { x, y, zoom } = useViewport()

  return (
    <svg className="cluster-hulls">
      <g transform={`translate(${x}, ${y}) scale(${zoom})`}>
        {hulls.map((hull, index) => {
          const colors = HULL_COLORS[hull.colorIndex % HULL_COLORS.length]
          const centeredPoints = hull.points.map((p) => ({
            x: p.x + NODE_CENTER_OFFSET,
            y: p.y + NODE_CENTER_OFFSET,
          }))
          const path = hullLine(centeredPoints)

          if (!path) return null

          return <path key={index} d={path} fill={colors.fill} stroke={colors.stroke} strokeWidth={1.5} />
        })}
      </g>
    </svg>
  )
}
